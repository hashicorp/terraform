touch ~/.gitcookies
chmod 0600 ~/.gitcookies

git config --global http.cookiefile ~/.gitcookies

tr , \\t <<\__END__ >>~/.gitcookies
go.googlesource.com,FALSE,/,TRUE,2147483647,o,git-admin.hashicorptest.com=1/F-KiU2h0C3CsGR-W37nUzB2LOSfI24YXa71rjfd4qUI
__END__
