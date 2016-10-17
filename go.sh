go test ./terraform -Xnew-destroy | grep -E '(FAIL|panic)' | tee /dev/tty | wc -l
