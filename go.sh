go test ./terraform -Xnew-apply -Xnew-destroy | grep -E '(FAIL|panic)' | tee /dev/tty | wc -l
