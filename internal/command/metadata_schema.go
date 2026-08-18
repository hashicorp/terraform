// the metadata_schema command is similar to providers schemas, in that it
// returns schemas for providers found in state and configuration (defaults to
// both, can use either),  but it also includes provisioner schemas, can get
// schemas for providers or provisioners not in the local config (using any
// configured installation methods), and can filter results.
package command

type MetadataSchemaCommand struct {
	Meta
}

func (c *MetadataSchemaCommand) Help() string {
	return metadataSchemaCommandHelp
}

func (c *MetadataSchemaCommand) Synopsis() string {
	return "Show schemas for providers and provisioners"
}

func (c *MetadataSchemaCommand) Run(args []string) int {
	// parse args
	// get providers & provisioners from local config & state
	// switch -
	//   return all config and state
	//   return only config or state
	//   lookup specific provider by version (check if provider/version tuple already found)
	//   lookup specific provisioner
	// add filtering to results

	return 0 // good job
}

const metadataSchemaCommandHelp = `
Usage: terraform [global options] metadata schema -json

  Prints out a json representation of provider and provisioner schemas.
`
