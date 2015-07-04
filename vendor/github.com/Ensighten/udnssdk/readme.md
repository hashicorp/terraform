# udnssdk - A ultradns SDK for GoLang
## about

This is a golang 'SDK' for UltraDNS that I copapasta'd from weppos/dnsimple.
It seemed like an ideal donor since the use case is terraform.

## How It works:
    client := udnssdk.NewClient("username","password",udnssdk.DefaultTestBaseURL)

There is DefaultTestBaseURL and DefaultLiveBaseURL.
When you call NewClient, it performs the 'oauth2' authorization step.
The refresh token is saved, but not implemented.  It should ideally be an error
condition triggering a reauth and retry.  But since Terraform is the use case, this won't be an issue.

### RRSet Declaration

    type RRSet struct {
      OwnerName string `json:"ownerName"`
      RRType string `json:"rrtype"`
      TTL int `json:"ttl"`
      RData []string `json:"rdata"`
    }

###GetRRSets(DomainName, RecordName(leave blank for all), RecordType[A|CNAME|ANY])
      rrsets, resp, err := client.Zones.GetRRSets("domain.com","","ANY")
      rrsets, resp, err := client.Zones.GetRRSets("domain.com","www","ANY")
      rrsets, resp, err := client.Zones.GetRRSets("domain.com","","MX")
      rrsets, resp, err := client.Zones.GetRRSets("domain.com","www","SRV")




###CreateRRSet(DomainName, RRSet)
      rr1 := &udnssdk.RRSet{OwnerName: "test", RRType: "A", TTL: 300, RData: []string{"127.0.0.1"}}
      resp2,err2 := client.Zones.CreateRRSet("ensighten.com",*rr1)

###UpdateRRSet(DomainName, RRSet)
UpdateRRSet requires you to specify the complete RRSet for the update.  This implementation does not support PATCHing.

    rr1 := &udnssdk.RRSet{OwnerName: "test", RRType: "A", TTL: 300, RData: []string{"192.168.1.1"}}
    resp2,err2 := client.Zones.CreateRRSet("domain.com",*rr1)

###DeleteRRSet(DomainName, RRSet)
Delete RRSet only uses the ownerName and RRType values from the RRSet object.

    rr3 := &udnssdk.RRSet{OwnerName: "test", RRType: "A"} // This is permissible.
    resp3,err3 := client.RRSets.DeleteRRSet("domain.com",*rr3)

## Example Program

        package main
        // udnssdk - a golang sdk for the ultradns REST service.
        // based on weppos/dnsimple

        import (
                "fmt"
                "udnssdk"
        )


        func main() {
          client := udnssdk.NewClient("username","password",udnssdk.DefaultTestBaseURL)
          if client == nil {
            fmt.Printf("Fail")
          } else {
            fmt.Printf("Win\n")
            rrsets, resp, err := client.RRSets.GetRRSets("domain.com","test","ANY")
            fmt.Printf("%+v\n",rrsets)
            fmt.Printf("%+v\n",resp)
            fmt.Printf("%+v\n",err)
            fmt.Printf("------------------------\n")
            fmt.Printf("---- Create RRSet\n")
            rr1 := &udnssdk.RRSet{OwnerName: "test", RRType: "A", TTL: 300, RData: []string{"127.0.0.1}}
            resp2,err2 := client.RRSets.CreateRRSet("domain.com",*rr1)
            fmt.Printf("Resp2: %+v\n", resp2)
            fmt.Printf("Err2: %+v\n", err2)
            fmt.Printf("------------------------\n")
            fmt.Printf("------------------------\n")
            fmt.Printf("---- Update RRSet\n")
            fmt.Printf("------------------------\n")
            rr2 := &udnssdk.RRSet{OwnerName: "test", RRType: "A", TTL: 300, RData: []string{"127.0.0.2"}}
            resp3, err3 := client.RRSets.UpdateRRSet("domain.com",*rr2)
            fmt.Printf("Resp3: %+v\n", resp3)
            fmt.Printf("Err3: %+v\n", err3)
            fmt.Printf("------------------------\n")

            fmt.Printf("------------------------\n")
            fmt.Printf("---- Delete RRSet\n")
            fmt.Printf("------------------------\n")
            resp4,err4 := client.RRSets.DeleteRRSet("domain.com",*rr2)
            fmt.Printf("Resp4: %+v\n", resp4)
            fmt.Printf("Err4: %+v\n", err4)
            fmt.Printf("------------------------\n")

          }
        }


#thanks
* [weppo's dnsimple go sdk @ github](https://github.com/weppos/go-dnsimple)
* [pearkes dnsimple sdk (this one is used by terraform) @ github](https://github.com/pearkes/dnsimple)
* [terraform](http://terraform.io)
* [UltraDNS's various SDK's](https://github.com/ultradns)
