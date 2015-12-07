# Systemd Deployment

### Steps for Use:

1. Make sure to change the path to your `kala` binary in ExecStart (within the `kala.service` file) to where it is.
1. Make sure to change the port in `kala.service`'s ExecStart to what you want it to be. (Defaults to 8000)
1. Move `kala.service` to `/etc/systemd/system/`
1. Run `systemctl enable /etc/systemd/system/kala.service`
1. Run `systemctl start kala` and then `systemctl status kala` to verify that it is working properly.


### Resources

* [Getting Started with systemd](https://coreos.com/docs/launching-containers/launching/getting-started-with-systemd/)

