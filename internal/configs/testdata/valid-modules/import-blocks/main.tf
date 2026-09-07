terraform {
    required_providers {
        localname = {
            source = "hashicorp/random"
        }
        random = {
            source = "hashicorp/random"
        }
    }
}

provider "random" {
    alias = "thisone"
}

import {
    to = random_string.test1
    provider = localname
    id = "importlocalname"
}

import {
    to = random_string.test2
    provider = random.thisone
    id = "importaliased"
}