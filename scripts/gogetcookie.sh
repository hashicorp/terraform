touch ~/.gitcookies
chmod 0600 ~/.gitcookies

git config --global http.cookiefile ~/.gitcookies

tr , \\t <<\__END__ >>~/.gitcookies
go.googlesource.com,FALSE,/,TRUE,2147483647,o,git-admin.hashicorptest.com=1/5dMSZVNdQscVq3on5V38iBrG9sP2TYRlbj3TDMJHKEvoBxl_QW-zl-L7a8lk-FU-
__END__
