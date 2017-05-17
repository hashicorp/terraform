FROM		google/golang:stable
MAINTAINER	Guillaume J. Charmes <guillaume@charmes.net>
CMD		/tmp/a.out
ADD		.	  /src
RUN		cd /src && go build -o /tmp/a.out
