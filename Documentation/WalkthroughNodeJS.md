# Running My First Application Inside Unikernel

Lets walk through all the steps needed to run your NodeJS application inside OSv unikernel using
Capstan. This step-by-step tutorial assumes that you have Capstan already
[installed](Documentation/Installation.md), but you know absolutely nothing about how to use it.

OK, suppose I'm a web developer and I've developed some awesome NodeJS server. I ran it hundreds of times
on my local machine during the development, but now it's time to go live! Typically I would ask our
administrator guy to launch another Ubuntu VM on our OpenStack and I would provision it through ssh.
But I've heard a lot about unikernels that are consuming less resources and everything, so I want
to give it a try and deploy my NodeJS application in unikernel.

## Summary for the impatient
Checkout example application and install it's libraries:
```bash
$ git clone https://github.com/amirrajan/word-finder.git word-finder
$ cd word-finder
$ node -v # must be 4.x.x
$ npm install
```
Navigate to project root direcotry and create Capstan configuration files there:
```
$ capstan package init --name com.example.word-finder --title "Word Finder" --author "Lemmy"
$ cat > meta/run.yaml <<EOL
runtime: node
config_set:
   word_count:
      main: /start.js
      env:
         PORT: 4000
config_set_default: word_count
EOL
```
Then compose and run your unikernel:
```bash
capstan package compose com.example.word-finder
capstan run com.example.word-finder -f 4004:4000
```
Open up your browser, navigate to `http://localhost:4004` and start using the application that runs
in unikernel.


## STEP 0: Develop your application locally

Suppose we've developed [this](https://github.com/amirrajan/word-finder) NodeJS application and put
it on the GitHub. Let's clone it:
```bash
$ git clone https://github.com/amirrajan/word-finder.git word-finder
$ # PROJECT_ROOT points to our NodeJS project root
$ PROJ_ROOT="$(pwd)/word-finder"
```
and run it locally:
```bash
$ cd $PROJECT_ROOT
$ npm install
$ node start.js
Listening on port: 3000
```
We can then open browser and navigate to http://localhost:3000 and see that the web server works.
Great! But wait, that's running on our development machine, not inside unikernel. Use CTRL + C to
stop the server.

Just out of curiosity, lets count number of files in this NodeJS project:
```
$ find . -type f | wc -l
46104
```
Wait what? 46k? Yes. This project is rather simple, but it uses some fancy libraries that contain
a lot of files. Just pointing out here that our NodeJS server is not a small one - and yet it will
run in the unikernel. Excited yet?

## STEP 1: Add meta information for Capstan
Capstan will seek for some configuration files when you ask it to prepare a unikernel. More precisely,
there are two configuration files `meta/package.yaml` and `meta/run.yaml` and you need to prepare
them for each unikernel. Think of them as a recipe that Capstan needs in order to know what unikernel
will suit your needs best.

### a) prepare meta/package.yaml

First things first. We need to give some meaningful name to our future unikernel. Navigate to your
project diretory and use Capstan utility command to generate this file for you:
```bash
$ cd $PROJECT_ROOT
$ capstan package init --name com.example.word-finder --title "Word Finder" --author "Lemmy"
Initializing package in /home/miha/git-repos/word-finder/meta
```
There you go, a folder named `meta` was generated inside project directory containing a file named
`package.yaml`. That's where information regarding unikernel content is now stored. You needn't
modify anything in this file. Yet, you can take a look at what it contains:
```bash
$ cd $PROJECT_ROOT
$ cat meta/package.yaml
name: com.example.word-finder
title: Word Finder
author: Lemmy
```


### b) prepare meta/run.yaml

Great! The unikernel has a name. Now we need to somehow inform Capstan that those 46k files are
written for NodeJS runtime. We do this by running:
```bash
$ cd $PROJECT_ROOT
$ capstan runtime init --runtime node
meta/run.yaml stub successfully added to your package. Please customize it in editor.
```
As the last line suggests, a file `meta/run.yaml` was created with some stubs and now
has to be edited manually. Go ahead open it up. It looks like this:
```yaml
$ cd $PROJECT_ROOT
$ cat meta/run.yaml
runtime: node
config_set:
   ################################################################
   ### This is one configuration set (feel free to rename it).  ###
   ################################################################
   myconfig1:
      # REQUIRED
      # Filepath of the NodeJS entrypoint (where server is defined).
      # Note that package root will correspond to filesystem root (/) in OSv image.
      # Example value: /server.js
      main: <filepath>

      # OPTIONAL
      # Environment variables.
      # A map of environment variables to be set when unikernel is run.
      # Example value:  env:
      #                    PORT: 8000
      #                    HOSTNAME: www.myserver.org
      env:
         <key>: <value>

   # Add as many named configurations as you need

# OPTIONAL
# What config_set should be used as default.
# This value can be overwritten with --runconfig argument.
config_set_default: myconfig1
```
What we see is a bunch of comments that try to explain what input is needed from you. Don't get scared,
you only need to type a few words in here. Final content of the file that is needed
to run our NodeJS application (all comments were removed for clarity) is:
```yaml
# meta/run.yaml
runtime: node
config_set:
   word_count:
      main: /start.js
      env:
         PORT: 4000
config_set_default: word_count
```
This reads as follows:
*Our application needs to be run using NodeJS runtime with entrypoint file /start.js and environment
variable 'PORT' set to 4000*.
Note that we could specify multiple configuration sets here (each e.g. with different entrypoint
file or different environment variables) and could then easily switch between them just before
unikernel is run. But in this case we only provide one config_set and name it *word_count*.

That's it! Capstan has all the information needed and we can go and create the unikernel at last.

## STEP 3: Compose unikernel

Once our NodeJS application has been equipped with appropriate configuration files, Capstan is
able to create the unikernel. Execute:
```bash
$ cd $PROJECT_ROOT
$ capstan package compose com.example.word-finder --size 200MB
(1)  Resolved runtime into: node
(2)  Using named configuration: 'word_count'
(3)  Prepending 'node' runtime dependencies to dep list: [app.node-4.4.5]
(4)  Importing com.example.word-finder...
(5)  Importing into ...capstan/repository/com.example.word-finder/com.example.word-finder.qemu
(6)  Uploading files to ...capstan/repository/com.example.word-finder/com.example.word-finder.qemu...
(7)  Setting cmdline: --norandom --nomount --noinit /tools/mkfs.so; /tools/cpiod.so --prefix /zfs/zfs; /zfs.so set compression=off osv
(8)  Uploading files 49319 / 49319 [=============================================] 100.00 % 1m9s
(9)  All files uploaded
(10) Command line set to: --env=PORT=4000 node /start.js
```
Let's observe what the output says. First it confirms that runtime "node" was detected (what a surprise;)
and that named configuration named "word_count" is being used.

The output then in line (3) says that node runtime dependencies were prepended to the dependencies list.
Luckily, Capstan is smart enough to figure out that if we've told him that the 46k files are written for
NodeJS runtime, then precompiled package containing NodeJS runtime is required.

Line (5) tells where the resulting unikernel will be created. Wait what, `.qemu` extension?
Yup, the unikernel that we get is nothing but qemu image (oficially the format is called
QCOW2). Which is really great since you can just grab it as any other virtual machine image and run
it on hipervisor. That's one of the best Capstan features: it produces a ready-to-run VM image.

Then in line (8) you see a progress bar. Capstan has to upload all your project files and required
packages into the target unikernel and here you see how it's doing. Note that we're uploading
nearly 50k files so it takes around a minute to complete. If you later make a small change to your
application code you needn't upload 50k files again, but rather update the existing unikernel by
adding --update flag.

Finally, in line (10), Capstan tells you what command will unikernel be booted with by default\*. This
command is calculated based on the content of your meta/run.yaml. To be honest, this is the *only*
command that will ever be executed in unikernel. Aren't unikernels simple?

\* *It is possible to change the boot command even for the composed unikernel. See
[documentation](Documentation/generated/CLI.md#capstanrun)
for more details.*

## STEP 4: Run unikernel
Once we have unikernel composed, we can run it. There's a bunch of possibilities here - you needn't
use Capstan tool for this.  You can upload the `.qemu` to the OpenStack and run it there thru Horizon
dashboard. Or you can use local installation of qemu and run it. Let's pick the third option, Capstan utility
function:
```bash
$ capstan run com.example.word-finder -f 4004:4000
(1) Resolved runtime into: node
(2) Using named configuration: 'word_count'
(3) Created instance: com.example.word-finder
(4) Setting cmdline: --env=PORT=4000 node /start.js
(5) OSv v0.24-116-g73b38d8
(6) eth0: 192.168.122.15
(7) Listening on port: 4000
```
We passed port forwarding rule `-f 4004:4000` to make unikernel's port 4000 accessible from our
localhost:4004. Go ahead, open your browser and navigate to `http://localhost:4004`. There it is,
our NodeJS application, running inside OSv unikernel!

Line (1) and (2) inform you what runtime is set and what run configuration is used.
As we've mentioned earlier, it is possible to change the boot command even after the unikernel is
composed already. Capstan therefore evaluates your meta/run.yaml once again here and makes sure that
the correct boot command is set.

Line (3) informs you about your instances's name and line (4) shows what boot command was set. You
needn't care about boot command, but the information is printed anyway to feed your curiosity.

All that's prineted after line (4) is captured from unikernel's stdin and stderr. OSv unikernel always
prints its kernel version (line 5) and network configuration (line 6) followed by whatever your
application is printing to the stdin/stderr. Line (7) was, for example, produced by our NodeJS
application. Will it print anything more while the unikernel is running, the text will appear here.

Congratulations! You've composed and run your first OSv unikernel.

---

## Applying small changes
Our server has 50k files so it takes nearly a minute to upload them all. If we then notice that the
composed unikernel crashes when booted, just because we forgot to remove some stupid debug line, we
need to remove the line and wait another minute... Well, that's not true! Luckily,
`capstan package compose` command supports `--update` flag.

Suppose we've followed the ste-by-step guide above. In STEP 3 we've composed unikernel named
'com.example.word-finder' and in STEP 4 we've run it. The server logged:
```
Listening on port: 4000
```
to the console (line 7 at STEP 4). But wait! We actually wanted to log timestamp as well. Not a problem,
open up server.js file and add line:
```
// Log timestamp.
console.log('Timestamp: ' + new Date());
```
to the bottom. Then repeat STEP 3, but this time with --update flage:
```
$ capstan package compose com.example.word-finder --update
```
Notice how the unikernel was updated in no time. To verify that the timestamp is really logged, boot
the unikernel:
```
$ capstan run com.example.word-finder -f 4004:4000
...
Listening on port: 4000
Timestamp: Thu Jan 05 2017 14:20:49 GMT+0000 (GMT)
```
As you can see, the timestamp is being logged.




