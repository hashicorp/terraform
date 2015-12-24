#
# This script generates CSV output reporting on the API Coverage of Terraform's
# AWS Provider.
#
# In addition to Ruby, it depends on a properly configured Go development
# environment with both terraform and aws-sdk-go present.
#

require 'csv'
require 'json'
require 'pathname'

module APIs
  module Terraform
    def self.path
      @path ||= Pathname(`go list -f '{{.Dir}}' github.com/hashicorp/terraform`.chomp)
    end

    def self.called?(api, op)
      `git -C "#{path}" grep "#{api}.*#{op}" -- builtin/providers/aws | wc -l`.chomp.to_i > 0
    end
  end

  module AWS
    def self.path
      @path ||= Pathname(`go list -f '{{.Dir}}' github.com/aws/aws-sdk-go/aws`.chomp).parent
    end

    def self.api_json_files
      Pathname.glob(path.join('**', '*.normal.json'))
    end

    def self.each
      api_json_files.each do |api_json_file|
        json = JSON.parse(api_json_file.read)
        api = api_json_file.dirname.basename
        json["operations"].keys.each do |op|
          yield api, op
        end
      end
    end
  end
end

csv = CSV.new($stdout)
csv << ["API", "Operation", "Called in Terraform?"]
APIs::AWS.each do |api, op|
  csv << [api, op, APIs::Terraform.called?(api, op)]
end
