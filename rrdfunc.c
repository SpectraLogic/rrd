#include <stdlib.h>
#include <rrd.h>
#include <rrd_client.h>

char *rrdError() {
	char *err = NULL;
	if (rrd_test_error()) {
		// RRD error is local for thread so other gorutine can call some RRD
		// function in the same thread before we use C.GoString. So we need to
		// copy current error before return from C to Go. It need to be freed
		// after C.GoString in Go code.
		err = strdup(rrd_get_error());
		if (err == NULL) {
			abort();
		}
	}
	return err;
}

char *rrdCreate(const char *filename, unsigned long step, time_t start, int argc, const char **argv) {
	rrd_clear_error();
	rrd_create_r(filename, step, start, argc, argv);
	return rrdError();
}

char *rrdUpdate(const char *filename, const char *template, int argc, const char **argv) {
	rrd_clear_error();
	rrd_updatex_r(filename, template, RRD_SKIP_PAST_UPDATES, argc, argv);
	return rrdError();
}

char *rrdDaemonUpdate(char *daemon, const char *filename, const char *template, int argc, const char **argv) {
	rrd_clear_error();
	rrdc_connect(daemon);
	if (rrdc_is_connected(daemon)){
		rrdc_update(filename, argc, argv);
	} else {
		rrd_update_r(filename, template, argc, argv);
	}
	return rrdError();
}

char *rrdGraph(rrd_info_t **ret, int argc, char **argv) {
	rrd_clear_error();
	*ret = rrd_graph_v(argc, argv);
	return rrdError();
}

char *rrdInfo(rrd_info_t **ret, char *filename) {
	rrd_clear_error();
	*ret = rrd_info_r(filename);
	return rrdError();
}

char *rrdDaemonInfo(rrd_info_t **ret, char* daemon, char *filename) {
	rrd_clear_error();
    rrdc_connect(daemon);
    if (rrdc_is_connected(daemon)){
    	*ret = rrdc_info(filename);
    } else {
        *ret = rrd_info_r(filename);
    }
	return rrdError();
}

char *rrdDaemonFetch(int *ret, char *daemon, char *filename, const char *cf, time_t *start, time_t *end, unsigned long *step, unsigned long *ds_cnt, char ***ds_namv, double **data) {
    rrd_clear_error();
    rrdc_connect(daemon);
    if (rrdc_is_connected(daemon)){
        *ret = rrdc_fetch(filename, cf, start, end, step, ds_cnt, ds_namv, data);
    } else {
	    *ret = rrd_fetch_r(filename, cf, start, end, step, ds_cnt, ds_namv, data);
    }
    return rrdError();
}

char *rrdFetch(int *ret, char *filename, const char *cf, time_t *start, time_t *end, unsigned long *step, unsigned long *ds_cnt, char ***ds_namv, double **data) {
	rrd_clear_error();
	*ret = rrd_fetch_r(filename, cf, start, end, step, ds_cnt, ds_namv, data);
	return rrdError();
}

char *rrdDaemonFlush(int *ret, const char *daemon, const char *filename) {
	rrd_clear_error();
	*ret = rrdc_flush_if_daemon(daemon, filename);
	return rrdError();
}

char *rrdXport(int *ret, int argc, char **argv, int *xsize, time_t *start, time_t *end, unsigned long *step, unsigned long *col_cnt, char ***legend_v, double **data) {
	rrd_clear_error();
	*ret = rrd_xport(argc, argv, xsize, start, end, step, col_cnt, legend_v, data);
	return rrdError();
}

char *arrayGetCString(char **values, int i) {
	return values[i];
}
