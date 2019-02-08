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
	"regexp"
	"sort"
	"strconv"
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


type VM interface {
	Name() string
	Destroy(ctx context.Context) (*object.Task, error)
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
//var imageName = flag.String("image-name", "", "image name")
var regex = flag.String("regex", "-([\\.0-9]+)$", "image name regex")
var keep = flag.Int("keep", 3, "keep last n images")
var dryRun = flag.Bool("dry-run", true, "Don't actually run clean")

type template struct {
	version int
	name string
	ref VM
}

type templateList []*template

func (tl *templateList) toString() string {
	output := "["
	for i, t := range *tl {
		if i == len(*tl) - 1 {
			output += t.name
		} else {
			output += t.name + ", "
		}
	}
	output += "]"
	return output
}

func (t *template) toString() string {
	return t.name
}

func main() {
	flag.Parse()

	var templates templateList

	re := regexp.MustCompile(*regex)
	log.Printf("Using regexp: %s", re.String())

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

	var temp *template
	for _, t := range items {
		temp = getTemplate(re, t)
		if temp != nil {
			templates = append(templates, temp)
		}
	}

	sort.Sort(byVersion(templates))
	deleted := templates[:len(templates) - *keep]
	kept := templates[len(templates) - *keep:]
	log.Printf("Next machines seleced for deletion %s", deleted.toString())
	log.Printf("Next machines will be kept %s", kept.toString())
	if !*dryRun {
		for _, d := range deleted {
			log.Printf("Deleting virtual machine '%s'", d.name)
			_, err := d.ref.Destroy(ctx)
			if err != nil {
				log.Printf("During deleting of %s, error occue, %s", d.name, err)
			}
		}
	}
}

func getTemplate(re *regexp.Regexp, machine VM) *template {
	ver := re.FindStringSubmatch(machine.Name())
	if len(ver) <= 1 {
		return nil
	}
	t := &template{
		ref: machine,
		name: machine.Name(),
		version: 0,
	}

	i, err := strconv.Atoi(ver[len(ver)-1])
	if err == nil {
		t.version = i
	}
	return t
}

func NewClient(ctx context.Context, host string, username string, password string, insecureFlag bool) (*govmomi.Client, error){
	u, err := url.Parse(fmt.Sprintf("https://%s%s", host, vim25.Path))
	u.User = url.UserPassword(username, password)
	if err != nil {
		return nil, err
	}
	return govmomi.NewClient(ctx, u, insecureFlag)
}

type byVersion []*template

func (s byVersion) Len() int {
	return len(s)
}
func (s byVersion) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byVersion) Less(i, j int) bool {
	return s[i].version < s[j].version
}
