package main

import (
	"errors"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/nickschuch/marco-lib"
	"github.com/samalba/dockerclient"
	"gopkg.in/alecthomas/kingpin.v2"
)

const name = "docker"

var (
	cliMarco     = kingpin.Flag("marco", "The remote Marco backend.").Default("http://localhost:81").OverrideDefaultFromEnvar("MARCO_ECS_URL").String()
	cliEndpoint  = kingpin.Flag("endpoint", "The Docker endpoint.").Default("unix:///var/run/docker.sock").OverrideDefaultFromEnvar("DOCKER_HOST").String()
	cliPorts     = kingpin.Flag("ports", "The ports you wish to proxy.").Default("80,8080,2368,8983").OverrideDefaultFromEnvar("MARCO_DOCKER_PORTS").String()
	cliEnv       = kingpin.Flag("env", "The container environment variable that is used as a domain identifier.").Default("DOMAIN").OverrideDefaultFromEnvar("MARCO_DOCKER_ENV").String()
	cliFrequency = kingpin.Flag("frequency", "How often to push to Marco").Default("15").OverrideDefaultFromEnvar("MARCO_ECS_FREQUENCY").Int64()
)

func main() {
	kingpin.Parse()

	for {
		log.WithFields(log.Fields{
			"type": "started",
		}).Info("Started pushing data to Marco.")

		err := Push(*cliMarco)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "failed",
			}).Info(err)
		} else {
			log.WithFields(log.Fields{
				"type": "completed",
			}).Info("Successfully pushed data to Marco.")
		}

		time.Sleep(time.Duration(*cliFrequency) * time.Second)
	}
}

func Push(m string) error {
	var b []marco.Backend

	// Get a list of backends keyed by the domain.
	list, err := getList()
	if err != nil {
		return err
	}

	// Ensure we have a list to send.
	if len(list) <= 0 {
		return errors.New("Empty list of environments.")
	}

	// Convert into the objects required for a push to Marco.
	for d, l := range list {
		n := marco.Backend{
			Type:   name,
			Domain: d,
			List:   l,
		}
		b = append(b, n)
	}

	// Attempt to send data to Marco.
	err = marco.Send(b, *cliMarco)
	if err != nil {
		return err
	}

	return nil
}

func getList() (map[string][]string, error) {
	// These are the URL's (keyed by domain) that we will return.
	list := make(map[string][]string)

	dockerClient, err := dockerclient.NewDockerClient(*cliEndpoint, nil)
	if err != nil {
		return list, err
	}

	containers, err := dockerClient.ListContainers(false, false, "")
	if err != nil {
		return list, err
	}

	for _, c := range containers {
		container, _ := dockerClient.InspectContainer(c.Id)

		// We try to find the domain environment variable. If we don't have one
		// then we have nothing left to do with this container.
		envDomain := getContainerEnv(*cliEnv, container.Config.Env)
		if len(envDomain) <= 0 {
			continue
		}

		// Here we build the proxy URL based on the exposed values provided
		// by NetworkSettings. If a container has not been exposed, it will
		// not work. We then build a URL based on these exposed values and:
		//   * Add a container reference so we can perform safe operations
		//     in the future.
		//   * Add the built url to the proxy lists for load balancing.
		for portString, portObject := range container.NetworkSettings.Ports {
			port := getPort(portString)
			if strings.Contains(*cliPorts, port) {
				url := getProxyUrl(portObject)
				if url != "" {
					list[envDomain] = append(list[envDomain], url)
				}
			}
		}
	}

	return list, nil
}

func getContainerEnv(key string, envs []string) string {
	for _, env := range envs {
		if strings.Contains(env, key) {
			envValue := strings.Split(env, "=")
			return envValue[1]
		}
	}
	return ""
}

func getPort(exposed string) string {
	port := strings.Split(exposed, "/")
	return port[0]
}

func getProxyUrl(binding []dockerclient.PortBinding) string {
	// Ensure we have PortBinding values to build against.
	if len(binding) <= 0 {
		return ""
	}

	// Handle IP 0.0.0.0 the same way Swarm does. We replace this with an IP
	// that uses a local context.
	port := binding[0].HostPort
	ip := binding[0].HostIp
	if ip == "0.0.0.0" {
		ip = "127.0.0.1"
	}

	return "http://" + ip + ":" + port
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
