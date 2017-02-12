// Copyright (C) 2015 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/moul/anonuuid"
	"github.com/renstrom/fuzzysearch/fuzzy"
)

const (
	// CacheRegion permits to access at the region field
	CacheRegion = iota
	// CacheArch permits to access at the arch field
	CacheArch
	// CacheOwner permits to access at the owner field
	CacheOwner
	// CacheTitle permits to access at the title field
	CacheTitle
	// CacheMarketPlaceUUID is used to determine the UUID of local images
	CacheMarketPlaceUUID
	// CacheMaxfield is used to determine the size of array
	CacheMaxfield
)

// ScalewayCache is used not to query the API to resolve full identifiers
type ScalewayCache struct {
	// Images contains names of Scaleway images indexed by identifier
	Images map[string][CacheMaxfield]string `json:"images"`

	// Snapshots contains names of Scaleway snapshots indexed by identifier
	Snapshots map[string][CacheMaxfield]string `json:"snapshots"`

	// Volumes contains names of Scaleway volumes indexed by identifier
	Volumes map[string][CacheMaxfield]string `json:"volumes"`

	// Bootscripts contains names of Scaleway bootscripts indexed by identifier
	Bootscripts map[string][CacheMaxfield]string `json:"bootscripts"`

	// Servers contains names of Scaleway servers indexed by identifier
	Servers map[string][CacheMaxfield]string `json:"servers"`

	// Path is the path to the cache file
	Path string `json:"-"`

	// Modified tells if the cache needs to be overwritten or not
	Modified bool `json:"-"`

	// Lock allows ScalewayCache to be used concurrently
	Lock sync.Mutex `json:"-"`

	hookSave func()
}

const (
	// IdentifierUnknown is used when we don't know explicitly the type key of the object (used for nil comparison)
	IdentifierUnknown = 1 << iota
	// IdentifierServer is the type key of cached server objects
	IdentifierServer
	// IdentifierImage is the type key of cached image objects
	IdentifierImage
	// IdentifierSnapshot is the type key of cached snapshot objects
	IdentifierSnapshot
	// IdentifierBootscript is the type key of cached bootscript objects
	IdentifierBootscript
	// IdentifierVolume is the type key of cached volume objects
	IdentifierVolume
)

// ScalewayResolverResult is a structure containing human-readable information
// about resolver results. This structure is used to display the user choices.
type ScalewayResolverResult struct {
	Identifier string
	Type       int
	Name       string
	Arch       string
	Needle     string
	RankMatch  int
	Region     string
}

// ScalewayResolverResults is a list of `ScalewayResolverResult`
type ScalewayResolverResults []ScalewayResolverResult

// NewScalewayResolverResult returns a new ScalewayResolverResult
func NewScalewayResolverResult(Identifier, Name, Arch, Region string, Type int) (ScalewayResolverResult, error) {
	if err := anonuuid.IsUUID(Identifier); err != nil {
		return ScalewayResolverResult{}, err
	}
	return ScalewayResolverResult{
		Identifier: Identifier,
		Type:       Type,
		Name:       Name,
		Arch:       Arch,
		Region:     Region,
	}, nil
}

func (s ScalewayResolverResults) Len() int {
	return len(s)
}

func (s ScalewayResolverResults) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ScalewayResolverResults) Less(i, j int) bool {
	return s[i].RankMatch < s[j].RankMatch
}

// TruncIdentifier returns first 8 characters of an Identifier (UUID)
func (s *ScalewayResolverResult) TruncIdentifier() string {
	return s.Identifier[:8]
}

func identifierTypeName(kind int) string {
	switch kind {
	case IdentifierServer:
		return "Server"
	case IdentifierImage:
		return "Image"
	case IdentifierSnapshot:
		return "Snapshot"
	case IdentifierVolume:
		return "Volume"
	case IdentifierBootscript:
		return "Bootscript"
	}
	return ""
}

// CodeName returns a full resource name with typed prefix
func (s *ScalewayResolverResult) CodeName() string {
	name := strings.ToLower(s.Name)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`--+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	return fmt.Sprintf("%s:%s", strings.ToLower(identifierTypeName(s.Type)), name)
}

// FilterByArch deletes the elements which not match with arch
func (s *ScalewayResolverResults) FilterByArch(arch string) {
REDO:
	for i := range *s {
		if (*s)[i].Arch != arch {
			(*s)[i] = (*s)[len(*s)-1]
			*s = (*s)[:len(*s)-1]
			goto REDO
		}
	}
}

// NewScalewayCache loads a per-user cache
func NewScalewayCache(hookSave func()) (*ScalewayCache, error) {
	var cache ScalewayCache

	cache.hookSave = hookSave
	homeDir := os.Getenv("HOME") // *nix
	if homeDir == "" {           // Windows
		homeDir = os.Getenv("USERPROFILE")
	}
	if homeDir == "" {
		homeDir = "/tmp"
	}
	cachePath := filepath.Join(homeDir, ".scw-cache.db")
	cache.Path = cachePath
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		cache.Clear()
		return &cache, nil
	} else if err != nil {
		return nil, err
	}
	file, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &cache)
	if err != nil {
		// fix compatibility with older version
		if err = os.Remove(cachePath); err != nil {
			return nil, err
		}
		cache.Clear()
		return &cache, nil
	}
	if cache.Images == nil {
		cache.Images = make(map[string][CacheMaxfield]string)
	}
	if cache.Snapshots == nil {
		cache.Snapshots = make(map[string][CacheMaxfield]string)
	}
	if cache.Volumes == nil {
		cache.Volumes = make(map[string][CacheMaxfield]string)
	}
	if cache.Servers == nil {
		cache.Servers = make(map[string][CacheMaxfield]string)
	}
	if cache.Bootscripts == nil {
		cache.Bootscripts = make(map[string][CacheMaxfield]string)
	}
	return &cache, nil
}

// Clear removes all information from the cache
func (c *ScalewayCache) Clear() {
	c.Images = make(map[string][CacheMaxfield]string)
	c.Snapshots = make(map[string][CacheMaxfield]string)
	c.Volumes = make(map[string][CacheMaxfield]string)
	c.Bootscripts = make(map[string][CacheMaxfield]string)
	c.Servers = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// Flush flushes the cache database
func (c *ScalewayCache) Flush() error {
	return os.Remove(c.Path)
}

// Save atomically overwrites the current cache database
func (c *ScalewayCache) Save() error {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.hookSave()
	if c.Modified {
		file, err := ioutil.TempFile(filepath.Dir(c.Path), filepath.Base(c.Path))
		if err != nil {
			return err
		}
		defer file.Close()
		if err := json.NewEncoder(file).Encode(c); err != nil {
			os.Remove(file.Name())
			return err
		}

		if err := os.Rename(file.Name(), c.Path); err != nil {
			os.Remove(file.Name())
			return err
		}
	}
	return nil
}

// ComputeRankMatch fills `ScalewayResolverResult.RankMatch` with its `fuzzy` score
func (s *ScalewayResolverResult) ComputeRankMatch(needle string) {
	s.Needle = needle
	s.RankMatch = fuzzy.RankMatch(needle, s.Name)
}

// LookUpImages attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpImages(needle string, acceptUUID bool) (ScalewayResolverResults, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	var res ScalewayResolverResults
	var exactMatches ScalewayResolverResults

	if acceptUUID && anonuuid.IsUUID(needle) == nil {
		if fields, ok := c.Images[needle]; ok {
			entry, err := NewScalewayResolverResult(needle, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierImage)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	needle = regexp.MustCompile(`^user/`).ReplaceAllString(needle, "")
	// FIXME: if 'user/' is in needle, only watch for a user image
	nameRegex := regexp.MustCompile(`(?i)` + regexp.MustCompile(`[_-]`).ReplaceAllString(needle, ".*"))
	for identifier, fields := range c.Images {
		if fields[CacheTitle] == needle {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierImage)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			exactMatches = append(exactMatches, entry)
		}
		if strings.HasPrefix(identifier, needle) || nameRegex.MatchString(fields[CacheTitle]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierImage)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		} else if strings.HasPrefix(fields[CacheMarketPlaceUUID], needle) || nameRegex.MatchString(fields[CacheMarketPlaceUUID]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierImage)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches, nil
	}

	return removeDuplicatesResults(res), nil
}

// LookUpSnapshots attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpSnapshots(needle string, acceptUUID bool) (ScalewayResolverResults, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	var res ScalewayResolverResults
	var exactMatches ScalewayResolverResults

	if acceptUUID && anonuuid.IsUUID(needle) == nil {
		if fields, ok := c.Snapshots[needle]; ok {
			entry, err := NewScalewayResolverResult(needle, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierSnapshot)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	needle = regexp.MustCompile(`^user/`).ReplaceAllString(needle, "")
	nameRegex := regexp.MustCompile(`(?i)` + regexp.MustCompile(`[_-]`).ReplaceAllString(needle, ".*"))
	for identifier, fields := range c.Snapshots {
		if fields[CacheTitle] == needle {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierSnapshot)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			exactMatches = append(exactMatches, entry)
		}
		if strings.HasPrefix(identifier, needle) || nameRegex.MatchString(fields[CacheTitle]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierSnapshot)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches, nil
	}

	return removeDuplicatesResults(res), nil
}

// LookUpVolumes attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpVolumes(needle string, acceptUUID bool) (ScalewayResolverResults, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	var res ScalewayResolverResults
	var exactMatches ScalewayResolverResults

	if acceptUUID && anonuuid.IsUUID(needle) == nil {
		if fields, ok := c.Volumes[needle]; ok {
			entry, err := NewScalewayResolverResult(needle, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierVolume)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	nameRegex := regexp.MustCompile(`(?i)` + regexp.MustCompile(`[_-]`).ReplaceAllString(needle, ".*"))
	for identifier, fields := range c.Volumes {
		if fields[CacheTitle] == needle {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierVolume)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			exactMatches = append(exactMatches, entry)
		}
		if strings.HasPrefix(identifier, needle) || nameRegex.MatchString(fields[CacheTitle]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierVolume)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches, nil
	}

	return removeDuplicatesResults(res), nil
}

// LookUpBootscripts attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpBootscripts(needle string, acceptUUID bool) (ScalewayResolverResults, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	var res ScalewayResolverResults
	var exactMatches ScalewayResolverResults

	if acceptUUID && anonuuid.IsUUID(needle) == nil {
		if fields, ok := c.Bootscripts[needle]; ok {
			entry, err := NewScalewayResolverResult(needle, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierBootscript)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	nameRegex := regexp.MustCompile(`(?i)` + regexp.MustCompile(`[_-]`).ReplaceAllString(needle, ".*"))
	for identifier, fields := range c.Bootscripts {
		if fields[CacheTitle] == needle {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierBootscript)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			exactMatches = append(exactMatches, entry)
		}
		if strings.HasPrefix(identifier, needle) || nameRegex.MatchString(fields[CacheTitle]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierBootscript)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches, nil
	}

	return removeDuplicatesResults(res), nil
}

// LookUpServers attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpServers(needle string, acceptUUID bool) (ScalewayResolverResults, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	var res ScalewayResolverResults
	var exactMatches ScalewayResolverResults

	if acceptUUID && anonuuid.IsUUID(needle) == nil {
		if fields, ok := c.Servers[needle]; ok {
			entry, err := NewScalewayResolverResult(needle, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierServer)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	nameRegex := regexp.MustCompile(`(?i)` + regexp.MustCompile(`[_-]`).ReplaceAllString(needle, ".*"))
	for identifier, fields := range c.Servers {
		if fields[CacheTitle] == needle {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierServer)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			exactMatches = append(exactMatches, entry)
		}
		if strings.HasPrefix(identifier, needle) || nameRegex.MatchString(fields[CacheTitle]) {
			entry, err := NewScalewayResolverResult(identifier, fields[CacheTitle], fields[CacheArch], fields[CacheRegion], IdentifierServer)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			res = append(res, entry)
		}
	}

	if len(exactMatches) == 1 {
		return exactMatches, nil
	}

	return removeDuplicatesResults(res), nil
}

// removeDuplicatesResults transforms an array into a unique array
func removeDuplicatesResults(elements ScalewayResolverResults) ScalewayResolverResults {
	encountered := map[string]ScalewayResolverResult{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v].Identifier] = elements[v]
	}

	// Place all keys from the map into a slice.
	results := ScalewayResolverResults{}
	for _, result := range encountered {
		results = append(results, result)
	}
	return results
}

// parseNeedle parses a user needle and try to extract a forced object type
// i.e:
//   - server:blah-blah -> kind=server, needle=blah-blah
//   - blah-blah -> kind="", needle=blah-blah
//   - not-existing-type:blah-blah
func parseNeedle(input string) (identifierType int, needle string) {
	parts := strings.Split(input, ":")
	if len(parts) == 2 {
		switch parts[0] {
		case "server":
			return IdentifierServer, parts[1]
		case "image":
			return IdentifierImage, parts[1]
		case "snapshot":
			return IdentifierSnapshot, parts[1]
		case "bootscript":
			return IdentifierBootscript, parts[1]
		case "volume":
			return IdentifierVolume, parts[1]
		}
	}
	return IdentifierUnknown, input
}

// LookUpIdentifiers attempts to return identifiers matching a pattern
func (c *ScalewayCache) LookUpIdentifiers(needle string) (ScalewayResolverResults, error) {
	results := ScalewayResolverResults{}

	identifierType, needle := parseNeedle(needle)

	if identifierType&(IdentifierUnknown|IdentifierServer) > 0 {
		servers, err := c.LookUpServers(needle, false)
		if err != nil {
			return ScalewayResolverResults{}, err
		}
		for _, result := range servers {
			entry, err := NewScalewayResolverResult(result.Identifier, result.Name, result.Arch, result.Region, IdentifierServer)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			results = append(results, entry)
		}
	}

	if identifierType&(IdentifierUnknown|IdentifierImage) > 0 {
		images, err := c.LookUpImages(needle, false)
		if err != nil {
			return ScalewayResolverResults{}, err
		}
		for _, result := range images {
			entry, err := NewScalewayResolverResult(result.Identifier, result.Name, result.Arch, result.Region, IdentifierImage)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			results = append(results, entry)
		}
	}

	if identifierType&(IdentifierUnknown|IdentifierSnapshot) > 0 {
		snapshots, err := c.LookUpSnapshots(needle, false)
		if err != nil {
			return ScalewayResolverResults{}, err
		}
		for _, result := range snapshots {
			entry, err := NewScalewayResolverResult(result.Identifier, result.Name, result.Arch, result.Region, IdentifierSnapshot)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			results = append(results, entry)
		}
	}

	if identifierType&(IdentifierUnknown|IdentifierVolume) > 0 {
		volumes, err := c.LookUpVolumes(needle, false)
		if err != nil {
			return ScalewayResolverResults{}, err
		}
		for _, result := range volumes {
			entry, err := NewScalewayResolverResult(result.Identifier, result.Name, result.Arch, result.Region, IdentifierVolume)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			results = append(results, entry)
		}
	}

	if identifierType&(IdentifierUnknown|IdentifierBootscript) > 0 {
		bootscripts, err := c.LookUpBootscripts(needle, false)
		if err != nil {
			return ScalewayResolverResults{}, err
		}
		for _, result := range bootscripts {
			entry, err := NewScalewayResolverResult(result.Identifier, result.Name, result.Arch, result.Region, IdentifierBootscript)
			if err != nil {
				return ScalewayResolverResults{}, err
			}
			entry.ComputeRankMatch(needle)
			results = append(results, entry)
		}
	}
	return results, nil
}

// InsertServer registers a server in the cache
func (c *ScalewayCache) InsertServer(identifier, region, arch, owner, name string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fields, exists := c.Servers[identifier]
	if !exists || fields[CacheTitle] != name {
		c.Servers[identifier] = [CacheMaxfield]string{region, arch, owner, name}
		c.Modified = true
	}
}

// RemoveServer removes a server from the cache
func (c *ScalewayCache) RemoveServer(identifier string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	delete(c.Servers, identifier)
	c.Modified = true
}

// ClearServers removes all servers from the cache
func (c *ScalewayCache) ClearServers() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Servers = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// InsertImage registers an image in the cache
func (c *ScalewayCache) InsertImage(identifier, region, arch, owner, name, marketPlaceUUID string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fields, exists := c.Images[identifier]
	if !exists || fields[CacheTitle] != name {
		c.Images[identifier] = [CacheMaxfield]string{region, arch, owner, name, marketPlaceUUID}
		c.Modified = true
	}
}

// RemoveImage removes a server from the cache
func (c *ScalewayCache) RemoveImage(identifier string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	delete(c.Images, identifier)
	c.Modified = true
}

// ClearImages removes all images from the cache
func (c *ScalewayCache) ClearImages() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Images = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// InsertSnapshot registers an snapshot in the cache
func (c *ScalewayCache) InsertSnapshot(identifier, region, arch, owner, name string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fields, exists := c.Snapshots[identifier]
	if !exists || fields[CacheTitle] != name {
		c.Snapshots[identifier] = [CacheMaxfield]string{region, arch, owner, name}
		c.Modified = true
	}
}

// RemoveSnapshot removes a server from the cache
func (c *ScalewayCache) RemoveSnapshot(identifier string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	delete(c.Snapshots, identifier)
	c.Modified = true
}

// ClearSnapshots removes all snapshots from the cache
func (c *ScalewayCache) ClearSnapshots() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Snapshots = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// InsertVolume registers an volume in the cache
func (c *ScalewayCache) InsertVolume(identifier, region, arch, owner, name string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fields, exists := c.Volumes[identifier]
	if !exists || fields[CacheTitle] != name {
		c.Volumes[identifier] = [CacheMaxfield]string{region, arch, owner, name}
		c.Modified = true
	}
}

// RemoveVolume removes a server from the cache
func (c *ScalewayCache) RemoveVolume(identifier string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	delete(c.Volumes, identifier)
	c.Modified = true
}

// ClearVolumes removes all volumes from the cache
func (c *ScalewayCache) ClearVolumes() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Volumes = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// InsertBootscript registers an bootscript in the cache
func (c *ScalewayCache) InsertBootscript(identifier, region, arch, owner, name string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	fields, exists := c.Bootscripts[identifier]
	if !exists || fields[CacheTitle] != name {
		c.Bootscripts[identifier] = [CacheMaxfield]string{region, arch, owner, name}
		c.Modified = true
	}
}

// RemoveBootscript removes a bootscript from the cache
func (c *ScalewayCache) RemoveBootscript(identifier string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	delete(c.Bootscripts, identifier)
	c.Modified = true
}

// ClearBootscripts removes all bootscripts from the cache
func (c *ScalewayCache) ClearBootscripts() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Bootscripts = make(map[string][CacheMaxfield]string)
	c.Modified = true
}

// GetNbServers returns the number of servers in the cache
func (c *ScalewayCache) GetNbServers() int {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	return len(c.Servers)
}

// GetNbImages returns the number of images in the cache
func (c *ScalewayCache) GetNbImages() int {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	return len(c.Images)
}

// GetNbSnapshots returns the number of snapshots in the cache
func (c *ScalewayCache) GetNbSnapshots() int {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	return len(c.Snapshots)
}

// GetNbVolumes returns the number of volumes in the cache
func (c *ScalewayCache) GetNbVolumes() int {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	return len(c.Volumes)
}

// GetNbBootscripts returns the number of bootscripts in the cache
func (c *ScalewayCache) GetNbBootscripts() int {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	return len(c.Bootscripts)
}
