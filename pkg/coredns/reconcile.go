package coredns

func ScheduleReconcile() {
	// Read corefile, ingressHostFile,
	// and make sure it's consistent against
	// our known ingresses
}

func AddIngress(hostname string, ip string) {
	hosts[hostname] = ip
	ScheduleReconcile()
}

func RemoveIngress(hostname string) {
	delete(hosts, hostname)
	ScheduleReconcile()
}
