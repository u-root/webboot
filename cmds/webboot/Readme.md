### Usage
The webboot program would do the following:
 - Present a menu with the existing cached distro options
 - If the user wants a distro that is not cached, they can download an ISO 
 - After the user decides on an ISO, boot it.

### Test
Our UI uses a package called Termui. Termui will parse the standard input into keyboard events and insert them into a channel, then from which the Termui get it's input.  For implement a unattended test, I manually build a series of keyboard events that reperesent my intented input for test, and insert them into a channel. Then I replace the original input channel with my channel in the test. So the go test could run a test of ui automatically.

See TestDownloadOption for an example:
 - create a channel by make(chan ui.Event).
 - use go pressKey(uiEvents, input) to translate the intented test input to keyboard events and push them to the uiEvents chanel.
 - use the uiEvents channel by call downloadOption.exec(uiEvents). (Main function will always call ui.PollEvents() to get the sandard input channel) 
 - all functions involving in ui input will provide a argument to indicate the input chanel.

 ### Hint
 If want to set up a cached directory in side the USB stick, the file structure of USB stick should be
+-- USB root
|  +-- Images (<--- the cache directory. It must be named as "Images")
|     +-- subdirectories or iso files
...