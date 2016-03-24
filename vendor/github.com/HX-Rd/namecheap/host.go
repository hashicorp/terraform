package namecheap

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var allowedRecordTypes = [...]string{"A", "AAAA", "CNAME", "MX", "MXE", "TXT", "URL", "URL301", "FRAME"}

const minTTL int = 60
const maxTTL int = 60000

func (c *Client) SetHosts(domain string, records []Record) ([]Record, error) {
	var ret RecordsCreateResult
	var domainSplit = strings.Split(domain, ".")

	if len(domainSplit) != 2 {
		return nil, fmt.Errorf("Domain does not contain SLD and TLD")
	}

	var numberOfRecords = len(records)
	params := map[string]string{
		"Command": "namecheap.domains.dns.setHosts",
		"SLD":     domainSplit[0],
		"TLD":     domainSplit[1],
	}
	itr := 0
	for itr < numberOfRecords {
		var sNumb = strconv.Itoa(itr + 1)
		params["HostName"+sNumb] = records[itr].HostName
		recordType := records[itr].RecordType
		if !CheckRecordType(recordType) {
			return nil, fmt.Errorf("Invalid record type")
		}
		params["RecordType"+sNumb] = recordType
		params["Address"+sNumb] = records[itr].Address
		params["MXPref"+sNumb] = strconv.Itoa(records[itr].MXPref)
		if records[itr].TTL < minTTL || records[itr].TTL > maxTTL {
			return nil, fmt.Errorf("Invalid ttl value")
		}
		params["TTL"+sNumb] = strconv.Itoa(records[itr].TTL)
		itr += 1
	}
	req, err := c.NewRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = decode(resp.Body, &ret)
	if err != nil {
		return nil, err
	}
	if ret.CommandResponse.DomainDNSSetHostsResult.IsSuccess == false {
		var errorBuf bytes.Buffer
		for _, responseError := range ret.Errors {
			errorBuf.WriteString("Number: ")
			errorBuf.WriteString(responseError.Number)
			errorBuf.WriteString(" Message: ")
			errorBuf.WriteString(responseError.Message)
			errorBuf.WriteString("\n")
		}
		return nil, fmt.Errorf(errorBuf.String())
	}
	newRecords, err := c.GetHosts(domain)
	if err != nil {
		return nil, err
	}
	return newRecords, nil
}

// GetRecords retrieves all the records for the given domain.
func (c *Client) GetHosts(domain string) ([]Record, error) {
	var recordsResponse RecordsResponse
	var domainSplit = strings.Split(domain, ".")
	params := map[string]string{
		"Command": "namecheap.domains.dns.getHosts",
		"SLD":     domainSplit[0],
		"TLD":     domainSplit[1],
	}
	req, err := c.NewRequest(params)
	if err != nil {
		return nil, err
	}
	resp, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = decode(resp.Body, &recordsResponse)
	if err != nil {
		return nil, err
	}
	return recordsResponse.CommandResponse.Records, nil
}

func CheckRecordType(recordType string) bool {
	for _, legalRecordType := range allowedRecordTypes {
		if recordType == legalRecordType {
			return true
		}
	}
	return false
}
