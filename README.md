# BrigDrake

[![codecov](https://codecov.io/gh/lovethedrake/brigdrake/branch/master/graph/badge.svg)](https://codecov.io/gh/lovethedrake/brigdrake)

__BrigDrake__ provides Drake pipeline support for
[Brigade](https://brigade.sh/).

## THIS PROJECT HIGHLY VOLATILE!

brigdrake implements the highly volatile
[DrakeSpec](https://github.com/lovethedrake/drakespec) and, as such is, itself,
highly volatile. Users are warned that breaking changes to this software are
likely at any point up until its eventual 1.0 release.

# Installation

Currently, BrigDrake recommends creating a _new_ Brigade instance that is
separate from any existing Brigade instance you may already have. This is
because BrigDrake requires specific Brigade configuration that would not be
conducive to building traditional, `brigade.js`-based Brigade projects.

The BrigDrake installation process is somewhat involved, but this is owed to the
complexity of creating a new GitHub App and isn't attributable to Brigade or
BrigDrake itself.

The installation process takes this general shape:

1. Install the `brigdrake` Helm chart.
2. Create a GitHub "app."
3. Update `brigdrake` with details generated during the GitHub app creation
   process.

The next few sections cover each of these in greater detail.

## Prerequisites

This documentation assumes you're installing BrigDrake into a Kuberntes cluster
that is capable of provisioning internet-facing load balancers. i.e. It is
easiest to follow along using a cluster deployed to one of the public clouds.

__Do not expect these instructions to work with minikube.__

## Install the Helm Chart

This documentation counts a detailed Helm overview as out of scope, so some
basic Helm knowledge is assumed.

Start by adding the `brigdrake` chart repository to Helm:

```console
$ helm repo add brigdrake https://lovethedrake.github.io/brigdrake
```

Proceed by installing the `brigdrake` chart. Initial installation can usually be
carried out using the chart's default configuration. We'll re-configure it later
once critical values are known.

```console
$ helm install brigdrake/brigdrake --name brigdrake --namespace brigdrake
```

Post-installation, wait for the `brigdrake-brigade-github-app` service in the
`brigdrake` namespace to obtain an external IP. Take note of this IP. You will
it in subsequent steps.

```console
$ kubectl get service brigdrake-brigade-github-app -n brigdrake
```

## Create a GitHub "App"

Proceed to the settings page for the GitHub user or organization that owns the
repositories you wish to build using BrigDrake. A menu on the left should
contain a __Developer settings__ sub-section. The options in that section may
be different depending on whether you're viewing settings for a user or an
organization, but in either case, drill down into this section until you find
an option to create a new __Github App__. (Do _not_ create a new OAuth App. This
is a different thing.)

1. Select a __GitHub App Name__. This name must be globally unique. When the
   form is submitted, validation will inform you if the name you have selected
   is already in use by another Github App.

1. The value of the __Homepage URL__ field is largely unimportant. It can be set
   to any valid URL. Consider using the GitHub user or organization's GitHub
   page.

1. Similarly, the __User authorization callback URL__ will be unused in our
   scenario and can be set to any valid URL. Again, consider using the GitHub
   user or organization's GitHub page.

1. __Webhook URL__ should incorporate the external IP of the
   `brigdrake-brigade-github-app` service (see previous section) and take the
   following form: `http://<IP>/events/github`.

1. __Webhook secret__ should be a strong password. Take note of whatever value
  you use. You will need it in subsequent steps.

1. Set the following permissions:
    - __Checks:__ Read & write
        - Only needed if you want job statuses reported back to GitHub. This
          requires additional project-level setup that isn't currently
          documented.
    - __Repository contents:__ Read-only
    - __Repository metadata:__ Read-only
    - __Pull requests:__ Read-only
    
1. Subscribe to the following events:
    - __Pull request__
    - __Push__

1. Submit the form.

1. When the App has been successfully created, you will be taken to a
   confirmation page.

    1. The page will prompt you to generate a private key. Follow the link to do
       so and download the key. __Note this is your only opportunity to do so.
       You cannot retrieve the key later; you can only replace it with a new
       one.__

## Update the BrigDrake Installation

Now that the GitHub App has been created, GitHub users, GitHub organizations, or
individual GitHub repositories that install/enable that app will securely
transmit events to BrigDrake's `brigade-github-app` gateway. This gateway,
requires some re-configuration with details from the previous section in order
to decrypt those secure payloads.

Extract the full set up configuration options from the ` brigdrake` chart and
save them to a file:

```console
$ helm inspect values brigdrake/brigdrake > my-brigdrake-values.yaml
```

Open the `my-brigdrake-values.yaml` in your favorite editor and find the
`brigade-github-app` section. Under that section is another named `github`.
Here, you will override three default values with details from the previous
section.

* __key__ must be configured with the entire contents of the private key file
  downloaded in the previous section. Pay special attention to syntax and
  indentation when setting the value for this field. (See example below.)

* __defaultSharedSecret__ must be set to the same shared secret we used when
  creating the GitHub App.

Example:

```yaml
  brigade-github-app:
    # ...
    github:
      key: |
        -----BEGIN RSA PRIVATE KEY-----
        MIIEpQIBAAKCAQEA/O4FNQYB2boyyO++cGMSawSzoKGNCIZ0J3NiS+y+CIsK7EXn
        mskOJY3.........................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ........................... This is a ..........................
        ........................... fake key. ..........................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ................................................................
        ....................................RfVw9nXXvmgyT0=
        -----END RSA PRIVATE KEY-----
      # ...
      defaultSharedSecret: UXn+lZdEvLXpPCQOfoRR
```

Now update the `brigdrake` installation with this new configuration:

```console
$ helm upgrade brigdrake brigdrake/brigdrake --values my-brigdrake-values.yaml
```

# Use

After following all the installation instructions, GitHub users, GitHub
organizations, or individual GitHub repositories that install/enable the GitHub
App will securely transmit events to your installation of BrigDrake.

Ultimately, BrigDrake is just __Brigade__ pre-configured with Drake pipeline
support, so a Brigade project must be created for any project you wish to build
using BrigDrake. Absent corresponding project configuration, Brigade (and thus,
brigdrake) will not respond to any of the events it receives.

To create a Brigade project, use the `brig` tool. Pre-built binaries for common
OSes and CPU architectures can be found on the
[Brigade releases page](https://github.com/brigadecore/brigade/releases).

```console
$ brig project create --namespace brigdrake
```

This is an interactive process. Follow the prompts. Default options and values
should be acceptable for nearly all fields, __but when prompted for the shared
secret source, select `Leave undefined`.__ This will ensure the default shared
secret for the entire Brigade instance is used.

With the project created, related events emitted from GitHub should trigger
Drake pipelines according to triggers defined in the project's `Drakefile.yaml`.

For an example `Drakefile.yaml`, refer to the
[brigdrake-demo](https://github.com/lovethedrake/brigdrake-demo) project. If
desired, you can fork that repository and push commits to the fork to see
BrigDrake in action.

Once triggered, a Drake pipeline built by BrigDrake is little different from any
typical Brigade build. All knowledge of Brigade and associated tools carries
over to BrigDrake.

These instructions count a complete overview of Brigade as out of scope, but do
check out Kashti (the Brigade UI) to see events and the builds and pipeline
executions they triggered. You can get the IP for Kashti as follows:

```console
$ kubectl get service brigdrake-kashti -n brigdrake
```

## Limitations

At present, BrigDrake only integrates with GitHub. i.e. Pipeline execution can
only be triggered, at this time, by events emitted by GitHub and received by
Brigade via the
[brigade-github-app](https://github.com/brigadecore/brigade-github-app) gateway.

This may change in the near future as both the BrigDrake project and the
[DrakeSpec](https://github.com/lovethedrake/drakespec) mature.

## Contributing

This project accepts contributions via GitHub pull requests. The
[Contributing](CONTRIBUTING.md) document outlines the process to help get your
contribution accepted.

## Code of Conduct

Although not a CNCF project, this project abides by the
[CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
