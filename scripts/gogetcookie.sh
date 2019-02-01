touch ~/.gitcookies
chmod 0600 ~/.gitcookies

git config --global http.cookiefile ~/.gitcookies

tr , \\t <<\__END__ >>~/.gitcookies
go.googlesource.com,FALSE,/,TRUE,2147483647,o,git-admin.hashicorptest.com=1/ba6U_xcdflHTPlB4ScWE5O63YMMlvbYtHxP8M_yufDs5-YdA8pqbXQZAtKwT7ROb
__END__
