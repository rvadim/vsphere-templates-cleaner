package main

import (
	"context"
	"flag"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"fmt"
	"github.com/vmware/govmomi/vim25"
)

type config struct {
	host string
	username string
	password string
	datacenter string
	insecureFlag bool
}

// getEnvString returns string from environment variable.
func getEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	return r
}

// getEnvBool returns boolean from environment variable.
func getEnvBool(v string, def bool) bool {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	switch strings.ToLower(r[0:1]) {
	case "t", "y", "1", "T":
		return true
	}

	return false
}

func getConfig() config {
	c := config{
		host: getEnvString("VSPHERE_HOST", ""),
		username: getEnvString("VSPHERE_USER", ""),
		password: getEnvString("VSPHERE_PASSWORD", ""),
		datacenter: getEnvString("VSPHERE_DC", ""),
		insecureFlag: getEnvBool("VSPHERE_INSECURE_CONNECTION", true),
	}
	if c.username == "" {
		c.username = getEnvString("VSPHERE_USERNAME", "")
	}

	if c.username == "" {
		log.Fatal("Error: env vars VSPHERE_USER or VSPHERE_USERNAME not specified")
	}

	if c.password == "" {
		log.Fatal("Error: env var VSPHERE_PASSWORD not specified")
	}

	if c.host == "" {
		log.Fatal("Error: env var VSPHERE_HOST not specified")
	}
	return c
}

//var imagePath = flag.String("image-path", "", "image path in folder view")
var imageName = flag.String("image-name", "", "image name")
var keep = flag.Int("keep", 3, "keep last n images")
var dryRun = flag.Bool("dry-run", true, "Don't actually run clean")


func main() {
	flag.Parse()
	//if *imagePath == "" {
	//	log.Fatal("Please specify image-path")
	//}
	if *imageName == "" {
		log.Fatal("Please specify image-name")
	}
	ctx := context.Background()
	c := getConfig()
	client, err := NewClient(ctx, c.host, c.username, c.password, c.insecureFlag)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Vsphere version: %v", client.Version)
	defer func() {
		err := client.Logout(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	finder := find.NewFinder(client.Client, false)
	datacenter, err := finder.DatacenterOrDefault(ctx, c.datacenter)
	if err != nil {
		log.Fatal(err)
	}
	finder.SetDatacenter(datacenter)

	items, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		log.Fatal(err)
	}

	templates := filterVMsByPrefix(items, *imageName)

	names := []string{}
	for _, i := range templates  {
		names = append(names, i.Name())
	}
	sort.Strings(names)
	if len(names) <= *keep {
		log.Printf("Next machines found: %v, keep: %v, nothing to do.", names, *keep)
		return
	}
	deleted := names[:len(names) - *keep]
	kept := names[len(names) - *keep:]
	log.Printf("Next machines seleced for deletion %v", deleted)
	log.Printf("Next machines will be kept %v", kept)
	if !*dryRun {
		for _, d := range deleted {
			for _, t := range templates {
				if t.Name() == d {
					log.Printf("Deleting virtual machine '%s'", d)
					_, err := t.Destroy(ctx)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func filterVMsByPrefix(vms []*object.VirtualMachine, prefix string) []*object.VirtualMachine{
	images := []*object.VirtualMachine{}
	for _, f := range vms {
		if strings.HasPrefix(f.Name(), *imageName) {
			images = append(images, f)
		}
	}
	return images
}

func NewClient(ctx context.Context, host string, username string, password string, insecureFlag bool) (*govmomi.Client, error){
	u, err := url.Parse(fmt.Sprintf("https://%s%s", host, vim25.Path))
	u.User = url.UserPassword(username, password)
	if err != nil {
		return nil, err
	}
	return govmomi.NewClient(ctx, u, insecureFlag)
}
