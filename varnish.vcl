# specify the VCL syntax version to use
vcl 4.1;

# import vmod_dynamic for better backend name resolution
import dynamic;

# we won't use any static backend, but Varnish still need a default one
backend default none;

# set up a dynamic director
# for more info, see https://github.com/nigoroll/libvmod-dynamic/blob/master/src/vmod_dynamic.vcc
sub vcl_init {
        new d = dynamic.director(port = "80");
}

acl purge {
    # "tasks.broadcaster";
    "10.0.0.0/8";
    "172.16.0.0/12";
    "192.168.0.0/16";
}

sub vcl_recv {
    if (req.method == "PURGE") {
        if (client.ip !~ purge) {
            return (synth(405, "Method Not Allowed"));
        }
        return (purge);
    }

	# force the host header to match the backend (not all backends need it,
	# but example.com does)
	set req.http.host = "wedos.cz";
	# set the backend
	set req.backend_hint = d.backend("wedos.cz");
}
