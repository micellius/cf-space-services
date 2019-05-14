// Copyright 2019 Vadim Tomnikov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

var isDebug = (os.Getenv("DEBUG") == "1")

// SpaceServicesPlugin - struct implementing the interface defined by the core CLI
type SpaceServicesPlugin struct{}

// GetMetadata - part of the plugin interface defined by the core CLI.
func (c *SpaceServicesPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "space-services-plugin",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "ss",
				HelpText: "List space services",
				UsageDetails: plugin.Usage{
					Usage: "ss\n   cf ss",
				},
			},
		},
	}
}

// Run - part of the plugin interface defined by the core CLI.
func (c *SpaceServicesPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "ss" {
		debug("Executing command ss")
		// Context
		debug("Getting current space")
		space, err := cliConnection.GetCurrentSpace()
		if err != nil {
			fmt.Printf("%s\n\n", colorize("FAILED", red, 1))
			fmt.Printf("Failed to get current space: %+v", err)
			return
		}
		debug("Getting access token")
		accessToken, err := cliConnection.AccessToken()
		if err != nil {
			fmt.Printf("%s\n\n", colorize("FAILED", red, 1))
			fmt.Printf("Failed to get access token: %+v", err)
			return
		}
		debug("Access token is: " + accessToken)
		debug("Getting api endpoint")
		apiEndpoint, err := cliConnection.ApiEndpoint()
		if err != nil {
			fmt.Printf("%s\n\n", colorize("FAILED", red, 1))
			fmt.Printf("Failed to get API endpoint: %+v", err)
			return
		}
		// Service Instances
		debug("Getting service instances")
		responseStrings, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/service_instances?q=space_guid:"+space.Guid+"&results-per-page=99")
		if err != nil {
			fmt.Printf("%s\n\n", colorize("FAILED", red, 1))
			fmt.Printf("Failed to fetch services: %+v", err)
			return
		}
		var responseObject CFResponse
		err = json.Unmarshal([]byte(strings.Join(responseStrings, "")), &responseObject)
		if err != nil {
			fmt.Printf("%s\n\n", colorize("FAILED", red, 1))
			fmt.Printf("Failed to unmarshal services: %+v", err)
			return
		}
		// Collect Service & Service Plan GUIDs, calculate longest service name
		debug("Collecting metadata")
		maxNameLength := 0
		maxServiceLength := 0
		maxServicePlanLength := 0
		serviceNamesMap := make(map[string]string)
		servicePlanNamesMap := make(map[string]string)
		readyChan := make(chan byte, len(responseObject.Resources))
		readyCount := 0
		for _, resource := range responseObject.Resources {
			if resource.Entity != nil {
				debug("Processing resource " + fmt.Sprintf("%+v", *resource.Entity.Name))
				maxNameLength = max(len(*resource.Entity.Name), maxNameLength)
				// Service names
				if resource.Entity.ServiceGUID != nil {
					if _, ok := serviceNamesMap[*resource.Entity.ServiceGUID]; !ok {
						serviceNamesMap[*resource.Entity.ServiceGUID] = ""
						readyCount--
						go func(ServiceGUID string) {
							debug("Getting metadata for service with GUID " + ServiceGUID)
							name, err := getName(accessToken, apiEndpoint+"/v2/services/"+ServiceGUID)
							debug("Received metadata for service with GUID " + ServiceGUID)
							if err != nil {
								fmt.Printf("Failed to get service metadata: %+v", err)
								readyChan <- 1
								readyCount++
								return
							}
							serviceNamesMap[ServiceGUID] = name
							maxServiceLength = max(len(serviceNamesMap[ServiceGUID]), maxServiceLength)
							readyChan <- 1
							readyCount++
						}(*resource.Entity.ServiceGUID)
					}
				} else {
					debug("No service GUID in " + *resource.Entity.Name)
				}
				// Service plan names
				if resource.Entity.ServicePlanGUID != nil {
					if _, ok := servicePlanNamesMap[*resource.Entity.ServicePlanGUID]; !ok {
						servicePlanNamesMap[*resource.Entity.ServicePlanGUID] = ""
						readyCount--
						go func(ServicePlanGUID string) {
							debug("Getting metadata for service plan with GUID " + ServicePlanGUID)
							name, err := getName(accessToken, apiEndpoint+"/v2/service_plans/"+ServicePlanGUID)
							debug("Received metadata for service plan with GUID " + ServicePlanGUID)
							if err != nil {
								fmt.Printf("Failed to get service plan metadata: %+v", err)
								readyChan <- 1
								readyCount++
								return
							}
							servicePlanNamesMap[ServicePlanGUID] = name
							maxServicePlanLength = max(len(servicePlanNamesMap[ServicePlanGUID]), maxServicePlanLength)
							readyChan <- 1
							readyCount++
						}(*resource.Entity.ServicePlanGUID)
					}
				} else {
					debug("No service plan GUID in " + *resource.Entity.Name)
				}
			}
		}
		// Wait service and service plan names are resolved
		debug("Waiting for " + strconv.Itoa(-readyCount) + " requests to finish")
		for readyCount < 0 {
			<-readyChan
			debug("Waiting for " + strconv.Itoa(-readyCount) + " requests to finish")
		}
		// Build service instances
		serviceInstances := make([]ServiceInstance, 0)
		for _, resource := range responseObject.Resources {
			if resource.Entity != nil && resource.Entity.Name != nil && resource.Entity.ServiceGUID != nil && resource.Entity.ServicePlanGUID != nil {
				serviceInstances = append(serviceInstances, ServiceInstance{
					Name:    *resource.Entity.Name,
					Service: serviceNamesMap[*resource.Entity.ServiceGUID],
					Plan:    servicePlanNamesMap[*resource.Entity.ServicePlanGUID],
				})
			}
		}
		// Sort by service and plan
		sort.Sort(ServiceInstances(serviceInstances))
		// Print
		cellSpacing := 3
		paddingName := "%-" + strconv.Itoa(maxNameLength+cellSpacing) + "v"
		paddingService := "%-" + strconv.Itoa(maxServiceLength+cellSpacing) + "v"
		paddingPlan := "%-" + strconv.Itoa(maxServicePlanLength+cellSpacing) + "v"
		fmt.Printf("%s\n\n", colorize("OK", green, 1))
		fmt.Printf("%s%s%s\n",
			colorize(fmt.Sprintf(paddingName, "name"), white, 1),
			colorize(fmt.Sprintf(paddingService, "service"), white, 1),
			colorize(fmt.Sprintf(paddingPlan, "plan"), white, 1))
		for _, instance := range serviceInstances {
			fmt.Printf("%s%s%s\n",
				fmt.Sprintf(paddingName, instance.Name),
				fmt.Sprintf(paddingService, instance.Service),
				fmt.Sprintf(paddingPlan, instance.Plan))
		}
	}
}

func main() {
	plugin.Start(new(SpaceServicesPlugin))
}

const (
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	magenta = 35
	cyan    = 36
	grey    = 37
	white   = 38
)

func colorize(message string, color uint, bold int) string {
	return fmt.Sprintf("\033[%d;%dm%s\033[0m", bold, color, message)
}

func debug(message string) {
	if isDebug {
		fmt.Println(message)
	}
}

// CFResponse Cloud Foundry response
type CFResponse struct {
	TotalResults int          `json:"total_results"`
	TotalPages   int          `json:"total_pages"`
	PrevURL      *string      `json:"prev_url,omitempty"`
	NextURL      *string      `json:"next_url,omitempty"`
	Resources    []CFResource `json:"resources"`
}

// CFResource Cloud Foundry response resource
type CFResource struct {
	// metadata
	Metadata *CFResourceMetadata `json:"metadata,omitempty"`

	// entity
	Entity *CFResourceEntity `json:"entity,omitempty"`
}

// CFResourceMetadata Cloud Foundry response resource metadata
type CFResourceMetadata struct {

	// created at
	CreatedAt string `json:"created_at,omitempty"`

	// guid
	GUID string `json:"guid,omitempty"`

	// updated at
	UpdatedAt string `json:"updated_at,omitempty"`

	// url
	URL string `json:"url,omitempty"`
}

// CFResourceEntity Cloud Foundry response resource entity
type CFResourceEntity struct {

	// name
	Name *string `json:"name,omitempty"`

	// label
	Label *string `json:"label,omitempty"`

	// service_guid
	ServiceGUID *string `json:"service_guid,omitempty"`

	// service_plan_guid
	ServicePlanGUID *string `json:"service_plan_guid,omitempty"`
}

// ServiceInstance - service instance
type ServiceInstance struct {
	Name    string
	Service string
	Plan    string
}

// ServiceInstances - array of service instances
type ServiceInstances []ServiceInstance

func (a ServiceInstances) Len() int      { return len(a) }
func (a ServiceInstances) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ServiceInstances) Less(i, j int) bool {
	return a[i].Service < a[j].Service ||
		(a[i].Service == a[j].Service && a[i].Plan < a[j].Plan) ||
		(a[i].Service == a[j].Service && a[i].Plan == a[j].Plan && a[i].Name < a[j].Name)
}

func max(arg0, arg1 int) int {
	if arg1 > arg0 {
		return arg1
	}
	return arg0
}

func getName(accessToken string, url string) (string, error) {
	result := ""
	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return result, err
	}
	request.Header.Add("Authorization", accessToken)
	debug("Making request to " + url)
	response, err := client.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		debug("ERROR response from " + url + " [" + response.Status + "]")
		return result, errors.New("Response returned error " + response.Status)
	}
	debug("OK response from " + url)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return result, err
	}
	var cfResponse CFResource
	err = json.Unmarshal(body, &cfResponse)
	if err != nil {
		return result, err
	}
	if cfResponse.Entity == nil {
		debug("Entity not found in resposne from " + url)
		return result, errors.New("Entity not found")
	}
	if cfResponse.Entity.Label != nil {
		debug("Label '" + *cfResponse.Entity.Label + "' found in resposne from " + url)
		result = *cfResponse.Entity.Label
	} else if cfResponse.Entity.Name != nil {
		debug("Name '" + *cfResponse.Entity.Name + "' found in resposne from " + url)
		result = *cfResponse.Entity.Name
	} else {
		debug("Neither name nor label found in response from " + url)
		result = ""
	}
	return result, nil
}
