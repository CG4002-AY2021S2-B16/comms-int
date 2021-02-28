echo "Resetting bluetooth 0/4"

sudo hciconfig hci0 down
echo "Resetting bluetooth 1/4"

sudo rmmod btusb
echo "Resetting bluetooth 2/4"

sudo modprobe btusb
echo "Resetting bluetooth 3/4"

sudo hciconfig hci0 up
echo "Resetting bluetooth completed"