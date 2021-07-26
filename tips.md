# Tips and Tricks

## Virtual Console

* Scroll up: <kbd>Shift</kbd> + <kbd>PgUp</kbd>
* Scroll down: <kbd>Shift</kbd> + <kbd>PgDown</kbd>
* History: `cat /dev/vcsu`
* You can get a copy of the text off the machine with:
    * `mkdir www`
    * `cd www`
    * `cp /dev/vcsu .`
    * `ip a` and take a note of the IP address.
    * `srvfiles -h 0.0.0.0 -p 80`
    * Then on the same LAN, open `http://192.168.x.y/vcsu` in a web browser.

## Setting up Vim for Go

Here are some quick instructions for installing https://github.com/fatih/vim-go

```
# First install the vim plugin manager:
curl -fLo ~/.vim/autoload/plug.vim --create-dirs https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim

# Add the vim-go plugin to the vimrc:
echo "call plug#begin('~/.vim/plugged')" >> ~/.vimrc
echo "Plug 'fatih/vim-go', { 'do': ':GoUpdateBinaries' }" >> ~/.vimrc
echo "call plug#end()" >> ~/.vimrc

# Install the plugins:
vim +PlugInstall
```

See the Run-It/Build-It/Fix-It/Test-It/... sections in
https://github.com/fatih/vim-go/wiki/Tutorial

You can setup shortcuts in your vimrc like this:

```
map <F3> :GoBuild<CR>
map <F4> :GoTest<CR>
```

## Running Go Debugger (Delve)

This depends on the previous steps to setup Go for Vim.

Adding this line to your vimrc makes the experience better:

```
echo "let g:go_debug_log_output = 0" >> ~/.vimrc
```

1. Start the debugger with `:GoDebugStart`.
2. Set a breakpoint on the current line with `:GoDebugBreakpoint` or `<F9>`.
3. Run to the breakpoint with `:GoDebugContinue` or `<F5>`.
4. You should see the following windows:
    * STACKTRACE: Hit enter on any of these to jump to the code.
    * VARIABLES: Local variables, arguments and registers
    * GOROUTINES: Hit enter on any of these to jump to the code.
    * OUTPUT: Output from the program and dlv
5. Use these commands to navigate your code (see `:help GoDebugStart`):
    * `:GoDebugNext` or `<F10>`: Run to next line in the current function
    * `:GoDebugStep` or `<F11>`: Run to next line which may be in another
      function (due to a function call).
    * `:GoDebugStepOut`: Run until the current function returns
    * `:GoDebugSet {var} {value}`: Only works for ints, bool, float and pointers.
    * `:GoDebugPrint {expr}`: Print the result of the expression.
    * `<F6>`: Print the value of word under the cursor.
6. Stop the debugger with `:GoDebugStop`.

## Dumping Stack Trace of Go Program

1. Use CTRL-Z to move the process to background.
2. Send the ABRT signal `kill -ABRT %1`
3. Resume the job with `fg`.
