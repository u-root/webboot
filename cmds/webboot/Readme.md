### Usage
The webboot program would do the following:
 - Present a menu with the existing cached distro options
 - If the user wants a distro that is not cached, they can download an ISO 
 - After the user decides on an ISO, boot it.

### Test
Our UI uses a package called Termui. Termui will parse the standard input into keyboard events and insert them into a channel, then from which the Termui get it's input.  For implement a unattended test, I manually build a series of keyboard events that reperesent my intented input for test, and insert them into a channel. Then I replace the original input channel with my channel in the test. So the go test could run a test of ui automatically.
