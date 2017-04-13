# .capstanignore

When composing a unikernel Capstan uploads **all** files from current directory to the unikernel. For
simple demo applications this is totally acceptable, but sooner or later you'll want to prevent
Capstan from uploading some subfolders (e.g. .idea/) and that's where `.capstanignore` comes in.


## Ignored by default
There are some files that are ignored (i.e. not copied to the unikernel that you're composing)
even if you don't provide your own `.capstanignore` file:
```
/meta/*
/mpm-pkg/*
/.git/*
```
These files do not get uploaded to the unikernel even if they exist in your project folder. Go ahead,
verify by running:
```bash
$ capstan config print
--- global configuration
CAPSTAN_ROOT: /home/miha/.capstan
CAPSTAN_REPO_URL: https://mikelangelo-capstan.s3.amazonaws.com/
CAPSTAN_DISABLE_KVM: false

--- curent directory configuration
CAPSTANIGNORE:
/meta/*
/mpm-pkg/*
/.git/*
```

## Specify your own .capstanignore
Go ahead, create a new file in your project root directory and name it `.capstanignore`. Below
please find a valid `.capstanignore` file example showing the syntax:

```
# ignores file 'myfile.txt' in project root directory
/myfile.txt

# ignores file 'myfile.txt' in '/myfolder' directory
/myfolder/myfile.txt

# ignores any file 'myfile.txt' in whole project (recursive)
/**/myfile.txt

# ignores folder 'myfolder' and all its content
/myfolder/*

# ignores any file with '.txt' suffix in project root directory
/*.txt

# ignores any file with '.txt' suffix in whole project (recursive)
/**/*.txt

```

As you can see the syntax is that of .gitignore only you need to start each pattern with slash `/`. Note
that negation (`!`) is not supported. You can see what files are actually getting excluded in your
case by using `--verbose` flag:
```bash
$ capstan package collect --verbose
Resolved runtime into: node
Prepending 'node' runtime dependencies to dep list: [eu.mikelangelo-project.app.node-4.4.5]
.capstanignore: ignore /bin
.capstanignore: ignore /bin/osv-launch-services.sh
.capstanignore: ignore /bin/osv-launch-worker.sh
.capstanignore: ignore /bin/upload_batch.sh
.capstanignore: ignore /doc
.capstanignore: ignore /doc/setup-phase.png
.capstanignore: ignore /doc/worker-phase.png
```
