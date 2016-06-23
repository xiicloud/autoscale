FROM scratch
ADD sample-config.json /etc/autoscale.json
ADD autoscale /bin/
CMD ["/bin/autoscale"]
