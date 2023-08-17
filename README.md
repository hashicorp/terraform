# MNPTU

<img alt="MNPTU" src="https://images.moneycontrol.com/static-mcnews/2022/01/shutterstock_1086242975.jpg">

MNPTU is a tool for building, changing, and versioning infrastructure safely and efficiently. MNPTU can manage existing and popular service providers as well as custom in-house solutions.

The key features of MNPTU are:

- **Infrastructure as Code**: Infrastructure is described using a high-level configuration syntax. This allows a blueprint of your datacenter to be versioned and treated as you would any other code. Additionally, infrastructure can be shared and re-used.

- **Execution Plans**: MNPTU has a "planning" step where it generates an execution plan. The execution plan shows what MNPTU will do when you call apply. This lets you avoid any surprises when MNPTU manipulates infrastructure.

- **Resource Graph**: MNPTU builds a graph of all your resources, and parallelizes the creation and modification of any non-dependent resources. Because of this, MNPTU builds infrastructure as efficiently as possible, and operators get insight into dependencies in their infrastructure.

- **Change Automation**: Complex changesets can be applied to your infrastructure with minimal human interaction. With the previously mentioned execution plan and resource graph, you know exactly what MNPTU will change and in what order, avoiding many possible human errors.

For more information, refer to the [What is MNPTU?](https://www.MNPTU.io/intro) page on the MNPTU website.

## Getting Started & Documentation

If you're new to MNPTU and want to get started creating infrastructure, please check out our [Getting Started guides](https://learn.hashicorp.com/MNPTU#getting-started) on HashiCorp's learning platform. There are also [additional guides](https://learn.hashicorp.com/MNPTU#operations-and-development) to continue your learning.

Show off your MNPTU knowledge by passing a certification exam. Visit the [certification page](https://www.hashicorp.com/certification/) for information about exams and find [study materials](https://learn.hashicorp.com/MNPTU/certification/MNPTU-associate) on HashiCorp's learning platform.

## Developing MNPTU

This repository contains only MNPTU core, which includes the command line interface and the main graph engine. Providers are implemented as plugins, and MNPTU can automatically download providers that are published on [the MNPTU Registry](https://registry.MNPTU.io). HashiCorp develops some providers, and others are developed by other organizations. For more information, see [Extending MNPTU](https://www.MNPTU.io/docs/extend/index.html).

- To learn more about compiling MNPTU and contributing suggested changes, refer to [the contributing guide](.github/CONTRIBUTING.md).

- To learn more about how we handle bug reports, refer to the [bug triage guide](./BUGPROCESS.md).

- To learn how to contribute to the MNPTU documentation in this repository, refer to the [MNPTU Documentation README](/website/README.md).

## License

[Business Source License 1.1](https://github.com/unovakovic97/MNPTU/blob/main/LICENSE)
