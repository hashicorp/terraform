.PHONY: all clean

all: .git/hooks/pre-commit
	go build .

clean:
	rm -f terraform-provider-nsone

.git/hooks/pre-commit:
	    if [ ! -f .git/hooks/pre-commit ]; then ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit; fi

