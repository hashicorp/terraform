.PHONY: all clean test

all: terraform-provider-nsone

terraform-provider-nsone:
	go build .

test:
	go test .../.

clean:
	rm terraform-provider-nsone

