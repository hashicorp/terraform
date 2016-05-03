package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/codegangsta/cli"
)

var api *cloudflare.API

// Map type used for printing a table
type table map[string]string

// Print a nicely-formatted table
func makeTable(zones []table, cols ...string) {
	// Store the maximum length of all columns
	// The default is the length of the title
	lens := make(map[string]int)
	for _, col := range cols {
		lens[col] = len(col)
	}
	// Increase the size of the column if it is larger than the current value
	for _, z := range zones {
		for col, val := range z {
			if _, ok := lens[col]; ok && len(val) > lens[col] {
				lens[col] = len(val)
			}
		}
	}
	// Print the headings and an underline for each heading
	for _, col := range cols {
		fmt.Printf("%s%s ", strings.Title(col), strings.Repeat(" ", lens[col]-len(col)))
	}
	fmt.Println()
	for _, col := range cols {
		fmt.Printf("%s ", strings.Repeat("-", lens[col]))
	}
	fmt.Println()
	// And finally print the table data
	for _, z := range zones {
		for _, col := range cols {
			fmt.Printf("%s%s ", z[col], strings.Repeat(" ", lens[col]-len(z[col])))
		}
		fmt.Println()
	}

}

func checkEnv() error {
	if api.APIKey == "" {
		return errors.New("API key not defined")
	}
	if api.APIEmail == "" {
		return errors.New("API email not defined")
	}
	return nil
}

// Utility function to check if CLI flags were given.
func checkFlags(c *cli.Context, flags ...string) error {
	for _, flag := range flags {
		if c.String(flag) == "" {
			cli.ShowSubcommandHelp(c)
			return fmt.Errorf("%s not specified", flag)
		}
	}
	return nil
}

func ips(*cli.Context) {
	ips, _ := cloudflare.IPs()
	fmt.Println("IPv4 ranges:")
	for _, r := range ips.IPv4CIDRs {
		fmt.Println(" ", r)
	}
	fmt.Println()
	fmt.Println("IPv6 ranges:")
	for _, r := range ips.IPv6CIDRs {
		fmt.Println(" ", r)
	}
}

func userInfo(*cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	user, err := api.UserDetails()
	if err != nil {
		fmt.Println(err)
		return
	}
	var output []table
	output = append(output, table{
		"ID":       user.ID,
		"Email":    user.Email,
		"Username": user.Username,
		"Name":     user.FirstName + " " + user.LastName,
		"2FA":      fmt.Sprintf("%t", user.TwoFA),
	})
	makeTable(output, "ID", "Email", "Username", "Name", "2FA")
}

func userUpdate(*cli.Context) {
}

func zoneList(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	zones, err := api.ListZones()
	if err != nil {
		fmt.Println(err)
		return
	}
	var output []table
	for _, z := range zones {
		output = append(output, table{
			"ID":     z.ID,
			"Name":   z.Name,
			"Plan":   z.Plan.LegacyID,
			"Status": z.Status,
		})
	}
	makeTable(output, "ID", "Name", "Plan", "Status")
}

func zoneInfo(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	var zone string
	if len(c.Args()) > 0 {
		zone = c.Args()[0]
	} else if c.String("zone") != "" {
		zone = c.String("zone")
	} else {
		cli.ShowSubcommandHelp(c)
		return
	}
	zones, err := api.ListZones(zone)
	if err != nil {
		fmt.Println(err)
		return
	}
	var output []table
	for _, z := range zones {
		output = append(output, table{
			"ID":           z.ID,
			"Zone":         z.Name,
			"Plan":         z.Plan.LegacyID,
			"Status":       z.Status,
			"Name Servers": strings.Join(z.NameServers, ", "),
			"Paused":       fmt.Sprintf("%t", z.Paused),
			"Type":         z.Type,
		})
	}
	makeTable(output, "ID", "Zone", "Plan", "Status", "Name Servers", "Paused", "Type")
}

func zonePlan(*cli.Context) {
}

func zoneSettings(*cli.Context) {
}

func zoneRecords(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	var zone string
	if len(c.Args()) > 0 {
		zone = c.Args()[0]
	} else if c.String("zone") != "" {
		zone = c.String("zone")
	} else {
		cli.ShowSubcommandHelp(c)
		return
	}

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create a an empty record for searching for records
	rr := cloudflare.DNSRecord{}
	var records []cloudflare.DNSRecord
	if c.String("id") != "" {
		rec, err := api.DNSRecord(zoneID, c.String("id"))
		if err != nil {
			fmt.Println(err)
			return
		}
		records = append(records, rec)
	} else {
		if c.String("name") != "" {
			rr.Name = c.String("name")
		}
		if c.String("content") != "" {
			rr.Name = c.String("content")
		}
		var err error
		records, err = api.DNSRecords(zoneID, rr)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	var output []table
	for _, r := range records {
		switch r.Type {
		case "MX":
			r.Content = fmt.Sprintf("%d %s", r.Priority, r.Content)
		case "SRV":
			dp := reflect.ValueOf(r.Data).Interface().(map[string]interface{})
			r.Content = fmt.Sprintf("%.f %s", dp["priority"], r.Content)
			// CloudFlare's API, annoyingly, automatically prepends the weight
			// and port into content, separated by tabs.
			// XXX: File this as a bug. LOC doesn't do this.
			r.Content = strings.Replace(r.Content, "\t", " ", -1)
		}
		output = append(output, table{
			"ID":      r.ID,
			"Type":    r.Type,
			"Name":    r.Name,
			"Content": r.Content,
			"Proxied": fmt.Sprintf("%t", r.Proxied),
			"TTL":     fmt.Sprintf("%d", r.TTL),
		})
	}
	makeTable(output, "ID", "Type", "Name", "Content", "Proxied", "TTL")
}

func dnsCreate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone", "name", "type", "content"); err != nil {
		return
	}
	zone := c.String("zone")
	name := c.String("name")
	rtype := c.String("type")
	content := c.String("content")
	ttl := c.Int("ttl")
	proxy := c.Bool("proxy")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	record := cloudflare.DNSRecord{
		Name:    name,
		Type:    strings.ToUpper(rtype),
		Content: content,
		TTL:     ttl,
		Proxied: proxy,
	}
	err = api.CreateDNSRecord(zoneID, record)
	if err != nil {
		fmt.Println("Error creating DNS record:", err)
	}
}

func dnsCreateOrUpdate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone", "name", "type", "content"); err != nil {
		return
	}
	zone := c.String("zone")
	name := c.String("name")
	rtype := strings.ToUpper(c.String("type"))
	content := c.String("content")
	ttl := c.Int("ttl")
	proxy := c.Bool("proxy")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Look for an existing record
	rr := cloudflare.DNSRecord{
		Name: name + "." + zone,
	}
	records, err := api.DNSRecords(zoneID, rr)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(records) > 0 {
		// Record exists - find the ID and update it.
		// This is imprecise without knowing the original content; if a label
		// has multiple RRs we'll just update the first one.
		for _, r := range records {
			if r.Type == rtype {
				rr.ID = r.ID
				rr.Type = r.Type
				rr.Content = content
				rr.TTL = ttl
				rr.Proxied = proxy
				err := api.UpdateDNSRecord(zoneID, r.ID, rr)
				if err != nil {
					fmt.Println("Error updating DNS record:", err)
				}
			}
		}
	} else {
		// Record doesn't exist - create it
		rr.Type = rtype
		rr.Content = content
		rr.TTL = ttl
		rr.Proxied = proxy
		err := api.CreateDNSRecord(zoneID, rr)
		if err != nil {
			fmt.Println("Error creating DNS record:", err)
		}
	}
}

func dnsUpdate(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone", "id"); err != nil {
		return
	}
	zone := c.String("zone")
	recordID := c.String("id")
	content := c.String("content")
	ttl := c.Int("ttl")
	proxy := c.Bool("proxy")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	record := cloudflare.DNSRecord{
		ID:      recordID,
		Content: content,
		TTL:     ttl,
		Proxied: proxy,
	}
	err = api.UpdateDNSRecord(zoneID, recordID, record)
	if err != nil {
		fmt.Println("Error updating DNS record:", err)
	}
}

func dnsDelete(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone", "id"); err != nil {
		return
	}
	zone := c.String("zone")
	recordID := c.String("id")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = api.DeleteDNSRecord(zoneID, recordID)
	if err != nil {
		fmt.Println("Error deleting DNS record:", err)
	}
}

func zoneCerts(*cli.Context) {
}

func zoneKeyless(*cli.Context) {
}

func zoneRailgun(*cli.Context) {
}

func pageRules(c *cli.Context) {
	if err := checkEnv(); err != nil {
		fmt.Println(err)
		return
	}
	if err := checkFlags(c, "zone"); err != nil {
		return
	}
	zone := c.String("zone")

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		fmt.Println(err)
		return
	}

	rules, err := api.ListPageRules(zoneID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%3s %-32s %-8s %s\n", "Pri", "ID", "Status", "URL")
	for _, r := range rules {
		var settings []string
		fmt.Printf("%3d %s %-8s %s\n", r.Priority, r.ID, r.Status, r.Targets[0].Constraint.Value)
		for _, a := range r.Actions {
			v := reflect.ValueOf(a.Value)
			var s string
			switch a.Value.(type) {
			case int:
				s = fmt.Sprintf("%s: %d", cloudflare.PageRuleActions[a.ID], v.Int())
			case float64:
				s = fmt.Sprintf("%s: %.f", cloudflare.PageRuleActions[a.ID], v.Float())
			case map[string]interface{}:
				vmap := a.Value.(map[string]interface{})
				s = fmt.Sprintf("%s: %.f - %s", cloudflare.PageRuleActions[a.ID], vmap["status_code"], vmap["url"])
			case nil:
				s = fmt.Sprintf("%s", cloudflare.PageRuleActions[a.ID])
			default:
				s = fmt.Sprintf("%s: %s", cloudflare.PageRuleActions[a.ID], strings.Title(strings.Replace(v.String(), "_", " ", -1)))
			}
			settings = append(settings, s)
		}
		fmt.Println("   ", strings.Join(settings, ", "))
	}
}

func railgun(*cli.Context) {
}

func main() {
	api = cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))

	app := cli.NewApp()
	app.Name = "flarectl"
	app.Usage = "CloudFlare CLI"
	app.Version = "2016.4.0"
	app.Commands = []cli.Command{
		{
			Name:    "ips",
			Aliases: []string{"i"},
			Action:  ips,
			Usage:   "Print CloudFlare IP ranges",
		},
		{
			Name:    "user",
			Aliases: []string{"u"},
			Usage:   "User information",
			Subcommands: []cli.Command{
				{
					Name:    "info",
					Aliases: []string{"i"},
					Action:  userInfo,
					Usage:   "User details",
				},
				{
					Name:    "update",
					Aliases: []string{"u"},
					Action:  userUpdate,
					Usage:   "Update user details",
				},
			},
		},

		{
			Name:    "zone",
			Aliases: []string{"z"},
			Usage:   "Zone information",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"l"},
					Action:  zoneList,
					Usage:   "List all zones on an account",
				},
				{
					Name:    "info",
					Aliases: []string{"i"},
					Action:  zoneInfo,
					Usage:   "Information on one zone",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
					},
				},
				{
					Name:    "plan",
					Aliases: []string{"p"},
					Action:  zonePlan,
					Usage:   "Plan information for one zone",
				},
				{
					Name:    "settings",
					Aliases: []string{"s"},
					Action:  zoneSettings,
					Usage:   "Settings for one zone",
				},
				{
					Name:    "dns",
					Aliases: []string{"d"},
					Action:  zoneRecords,
					Usage:   "DNS records for a zone",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
					},
				},
				{
					Name:    "railgun",
					Aliases: []string{"r"},
					Action:  zoneRailgun,
					Usage:   "Railguns for a zone",
				},
				{
					Name:    "certs",
					Aliases: []string{"c"},
					Action:  zoneCerts,
					Usage:   "Custom SSL certificates for a zone",
				},
				{
					Name:    "keyless",
					Aliases: []string{"k"},
					Action:  zoneKeyless,
					Usage:   "Keyless SSL for a zone",
				},
			},
		},

		{
			Name:    "dns",
			Aliases: []string{"d"},
			Usage:   "DNS records",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"l"},
					Action:  zoneRecords,
					Usage:   "List DNS records for a zone",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "id",
							Usage: "record id",
						},
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "record name",
						},
						cli.StringFlag{
							Name:  "content",
							Usage: "record content",
						},
					},
				},
				{
					Name:    "create",
					Aliases: []string{"c"},
					Action:  dnsCreate,
					Usage:   "Create a DNS record",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "record name",
						},
						cli.StringFlag{
							Name:  "type",
							Usage: "record type",
						},
						cli.StringFlag{
							Name:  "content",
							Usage: "record content",
						},
						cli.IntFlag{
							Name:  "ttl",
							Usage: "TTL (1 = automatic)",
							Value: 1,
						},
						cli.BoolFlag{
							Name:  "proxy",
							Usage: "proxy through CloudFlare (orange cloud)",
						},
					},
				},
				{
					Name:    "update",
					Aliases: []string{"u"},
					Action:  dnsUpdate,
					Usage:   "Update a DNS record",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
						cli.StringFlag{
							Name:  "id",
							Usage: "record id",
						},
						cli.StringFlag{
							Name:  "content",
							Usage: "record content",
						},
						cli.IntFlag{
							Name:  "ttl",
							Usage: "TTL (1 = automatic)",
							Value: 1,
						},
						cli.BoolFlag{
							Name:  "proxy",
							Usage: "proxy through CloudFlare (orange cloud)",
						},
					},
				},
				{
					Name:    "create-or-update",
					Aliases: []string{"o"},
					Action:  dnsCreateOrUpdate,
					Usage:   "Create a DNS record, or update if it exists",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "record name",
						},
						cli.StringFlag{
							Name:  "content",
							Usage: "record content",
						},
						cli.StringFlag{
							Name:  "type",
							Usage: "record type",
						},
						cli.IntFlag{
							Name:  "ttl",
							Usage: "TTL (1 = automatic)",
							Value: 1,
						},
						cli.BoolFlag{
							Name:  "proxy",
							Usage: "proxy through CloudFlare (orange cloud)",
						},
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"d"},
					Action:  dnsDelete,
					Usage:   "Delete a DNS record",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
						cli.StringFlag{
							Name:  "id",
							Usage: "record id",
						},
					},
				},
			},
		},

		{
			Name:    "pagerules",
			Aliases: []string{"p"},
			Usage:   "Page Rules",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"l"},
					Action:  pageRules,
					Usage:   "List Page Rules for a zone",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "zone",
							Usage: "zone name",
						},
					},
				},
			},
		},

		{
			Name:    "railgun",
			Aliases: []string{"r"},
			Usage:   "Railgun information",
			Action:  railgun,
		},
	}
	app.Run(os.Args)
}
