go test ./terraform | grep -E '(FAIL|panic)' | tee /dev/tty | wc -l
