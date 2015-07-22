.PHONY: all clean test

all: terraform-provider-nsone

terraform-provider-nsone:
	go build .

test:
	cd nsone ; go test -v .

clean:
	rm terraform-provider-nsone

