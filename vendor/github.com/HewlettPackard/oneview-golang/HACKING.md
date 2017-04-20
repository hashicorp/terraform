Acceptance test cases
---------------------
This guide will describe steps we're using on running acceptance test cases.

Environment setup
------------------
Typical scenario, `make build` and `make test` should work as normal for the
overall project.

craete an environment, `$HOME/.oneview.houston.tb.200.env`, script to export these values:

```bash
cat > "$HOME/.oneview.env" << ONEVIEW
export ONEVIEW_APIVERSION=120

export ONEVIEW_ILO_USER=docker
export ONEVIEW_ILO_PASSWORD=password

export ONEVIEW_ICSP_ENDPOINT=https://15.x.x.x
export ONEVIEW_ICSP_USER=username
export ONEVIEW_ICSP_PASSWORD=password
export ONEVIEW_ICSP_DOMAIN=LOCAL

export ONEVIEW_OV_ENDPOINT=https://15.x.x.x
export ONEVIEW_OV_USER=username
export ONEVIEW_OV_PASSWORD=password
export ONEVIEW_OV_DOMAIN=LOCAL

export ONEVIEW_I3S_ENDPOINT=https://15.x.x.x

export ONEVIEW_SSLVERIFY=true

ONEVIEW

```
Now you can setup environment value for the test cases you plan to run.

```bash
export TEST_CASES=EGSL_HOUSTB200_LAB:~/.oneview.houston.tb.200.env
```
Run the acceptance test
-------------------------
Acceptance test can be executed:

```bash
make test-acceptance
```

Running debug log output
-------------------------
Output from test case debugging log can be handy.

```bash
ONEVIEW_DEBUG=true make test-acceptance
```

Run a single specific test with docker
---------------------------
Sometimes it's usefull to run just a single test case.
```bash
TEST_CASES=EGSL_HOUSTB200_LAB:/home/docker/creds.env \
ONEVIEW_DEBUG=true \
   make test-case TEST_RUN='-test.run=TestGetAPIVersion'
```

Run a single test without docker
------------------------------
Setup the libraries, example:
```
cp -R vendor/* /home/docker/go/src/
ln -s /home/docker/git/github.com/HewlettPackard/oneview-golang /home/docker/go/src/github.com/HewlettPackard/oneview-golang
```

Run a Test
```bash
TEST_CASES=EGSL_HOUSTB200_LAB:/home/docker/creds.env \
USE_CONTAINER=false \
ONEVIEW_DEBUG=true \
   make test-acceptance TEST_RUN='-test.run=TestGetAPIVersion'
```

Updating external dependencies
------------------------------
This project is no relying on glide to provide reliable & repeatable builds.
To learn more about glide, please visit : https://glide.sh/

Special thanks to Matt Farina for introducing it to us.

Start by installing glide:

```
curl https://glide.sh/get | sh
```

1. Add a dependency by editing the glide.yml in the root directory or run glide,
   glide get <package>#<version>, for example:

   ```
   glide get github.com/docker/machine#0.8.0
   ```

2. To update the existing packages, we use glide install.  Edit the glide.yaml and run:

   ```
   make glide
   ```

3. Run a build in a docker container.

   ```
   make test
   ```

4. Evaluate changes.
   At this point you might have changes to the dependent libraries that have
   to be incorporated into the build process.   Update any additional or
   no longer libraries by editing the file : [glide.yaml](glide.yaml).  
   This file contains all needed packages.
   Whenever adjusting libraries, make sure to re-do steps 1-3 iteratively.

5. Ok, it all test and passes, so it's time to commit your changes.

  ```
  git add --all
  ```
  Use `git status` to review additions, removals, and changes.
  Use `git commit -s -m "library update version X.X"` to commit your changes.
