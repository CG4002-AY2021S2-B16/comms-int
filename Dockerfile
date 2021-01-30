FROM golang:1.16-rc

COPY . /var/www/comms-int
WORKDIR /var/www/comms-int/cmd
CMD ["go", "run", "main.go"]
