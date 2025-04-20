
# Changelog Entry for v1.13

## [Unreleased]
### Added
- modify cidr block in networks variable 

### Changed
- Updated the cidr block in networks variable website/docs/language/functions/flatten.mdx 

### Deprecated
- n/a

### Fixed
- fixed issue when run terraform plan
-  Error: expected cidr_block to contain a valid network Value, expected 10.1.0.0/16, got 10.1.1.0/16
│ 
│   with aws_vpc.example["public"],
│   on main.tf line 47, in resource "aws_vpc" "example":
│   47:   cidr_block = each.value.cidr_block
│ 
╵
╷
│ Error: expected cidr_block to contain a valid network Value, expected 10.1.0.0/16, got 10.1.2.0/16
│ 
│   with aws_vpc.example["dmz"],
│   on main.tf line 47, in resource "aws_vpc" "example":
│   47:   cidr_block = each.value.cidr_block


**Notes:**  
- n/a
