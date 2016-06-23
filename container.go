package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
)

type Container struct {
	Id     string
	Names  []string
	Labels map[string]string
	Status string
}

// Get a list of all the running containers of the given service.
func listContainers(app, service string) ([]*Container, error) {
	appFilter := fmt.Sprintf("csphere_instancename=%s", app)
	serviceFilter := fmt.Sprintf("csphere_servicename=%s", service)
	args := map[string][]string{
		"labels": {appFilter, serviceFilter},
	}

	u, err := url.Parse(controllerAddr)
	if err != nil {
		return nil, err
	}
	filter, _ := json.Marshal(args)
	u.Path = "/api/containers"
	v := url.Values{}
	v.Set("filter", string(filter))
	v.Set("ApiKey", apiKey)
	u.RawQuery = v.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	containers := []*Container{}
	err = json.NewDecoder(res.Body).Decode(&containers)
	if err != nil {
		return nil, err
	}

	r := make([]*Container, 0, len(containers))
	for _, c := range containers {
		if strings.HasPrefix(c.Status, "Up ") {
			r = append(r, c)
		}
	}

	logrus.Debugf("Get %d runnig containers", len(r))

	return r, nil
}

func scale(app, service string, n int) error {
	u := controllerAddr + fmt.Sprintf("/api/instances/%s/%s/changesum?ApiKey=%s&sum=%d", app, service, apiKey, n)
	logrus.Debugf("scale api url: %s", u)
	req, err := http.NewRequest("PATCH", u, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK && res.StatusCode != http.StatusNoContent {
		return nil
	}

	return fmt.Errorf("unexpecte http status %d", res.StatusCode)
}
