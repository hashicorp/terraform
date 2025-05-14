# Terraform Core GitHub Bug Triage & Labeling
The Terraform Core team has adopted a more structured bug triage process than we previously used. Our goal is to respond to reports of issues quickly.

When a bug report is filed, our goal is to either:
1. Get it to a state where it is ready for engineering to fix it in an upcoming Terraform release, or 
2. Close it and explain why, if we can't help

## Process

### 1. [Newly created issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Anew+label%3Abug+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3A%22waiting+for+reproduction%22+-label%3A%22waiting-response%22+-label%3Aexplained+) require initial filtering. 

These are raw reports that need categorization and support clarifying them. They need the following done:

* label backends, provisioners, and providers so we can route work on codebases we don't support to the correct teams
* point requests for help to the community forum and close the issue
* close reports against old versions we no longer support
* prompt users who have submitted obviously incomplete reproduction cases for additional information

If an issue requires discussion with the user to get it out of this initial state, leave "new" on there and label it "waiting-response" until this phase of triage is done.

Once this initial filtering has been done, remove the new label. If an issue subjectively looks very high-impact and likely to impact many users, assign it to the [appropriate milestone](https://github.com/hashicorp/terraform/milestones) to mark it as being urgent.

### 2. Clarify [unreproduced issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+created%3A%3E2020-05-01+-label%3Abackend%2Fk8s+-label%3Aprovisioner%2Fsalt-masterless+-label%3Adocumentation+-label%3Aprovider%2Fazuredevops+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent+-label%3Abackend%2Fmanta+-label%3Abackend%2Fatlas+-label%3Abackend%2Fetcdv3+-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3Anew+-label%3A%22waiting+for+reproduction%22+-label%3Awaiting-response+-label%3Aexplained+sort%3Acreated-asc+)

A core team member initially determines whether the issue is immediately reproducible. If they cannot readily reproduce it, they label it "waiting for reproduction" and correspond with the reporter to describe what is needed. When the issue is reproduced by a core team member, they label it "confirmed". 

"confirmed" issues should have a clear reproduction case. Anyone who picks it up should be able to reproduce it readily without having to invent any details.

Note that the link above excludes issues reported before May 2020; this is to avoid including issues that were reported prior to this new process being implemented. [Unreproduced issues reported before May 2020](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+created%3A%3C2020-05-01+-label%3Aprovisioner%2Fsalt-masterless+-label%3Adocumentation+-label%3Aprovider%2Fazuredevops+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent+-label%3Abackend%2Fmanta+-label%3Abackend%2Fatlas+-label%3Abackend%2Fetcdv3+-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3Anew+-label%3A%22waiting+for+reproduction%22+-label%3Awaiting-response+-label%3Aexplained+sort%3Areactions-%2B1-desc) will be triaged as capacity permits.


### 3. Explain or fix [confirmed issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+-label%3Aexplained+-label%3Abackend%2Foss+-label%3Abackend%2Fk8s+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+)
The next step for confirmed issues is to either:

* explain why the behavior is expected, label the issue as "working as designed", and close it, or
* locate the cause of the defect in the codebase. When the defect is located, and that description is posted on the issue, the issue is labeled "explained". In many cases, this step will get skipped if the fix is obvious, and engineers will jump forward and make a PR. 

 [Confirmed crashes](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Acrash+label%3Abug+-label%3Abackend%2Fk8s+-label%3Aexplained+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+) should generally be considered high impact

### 4. The last step for [explained issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+label%3Aexplained+no%3Amilestone+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+) is to make a PR to fix them. 

Explained issues that are expected to be fixed in a future release should be assigned to a milestone

## GitHub Issue Labels
label                    | description
------------------------ | -----------
new                      | new issue not yet triaged
explained                | a Terraform Core team member has described the root cause of this issue in code
waiting for reproduction | unable to reproduce issue without further information 
not reproducible         | closed because a reproduction case could not be generated
duplicate                | issue closed because another issue already tracks this problem
confirmed                | a Terraform Core team member has reproduced this issue
working as designed      | confirmed as reported and closed because the behavior is intended
pending project          | issue is confirmed but will require a significant project to fix

## Lack of response and unreproducible issues
When bugs that have been [labeled waiting response](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+-label%3Abackend%2Foss+-label%3Abackend%2Fk8s+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent+-label%3Abackend%2Fmanta+-label%3Abackend%2Fatlas+-label%3Abackend%2Fetcdv3+-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3A%22waiting+for+reproduction%22+label%3Awaiting-response+-label%3Aexplained+sort%3Aupdated-asc+) or [labeled "waiting for reproduction"](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent+-label%3Abackend%2Fmanta+-label%3Abackend%2Fatlas+-label%3Abackend%2Fetcdv3+-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+label%3A%22waiting+for+reproduction%22+-label%3Aexplained+sort%3Aupdated-asc+) for more than 30 days, we'll use our best judgement to determine whether it's more helpful to close it or prompt the reporter again. If they again go without a response for 30 days, they can be closed with a polite message explaining why and inviting the person to submit the needed information or reproduction case in the future.

The intent of this process is to get fix the maximum number of bugs in Terraform as quickly as possible, and having un-actionable bug reports makes it harder for Terraform Core team members and community contributors to find bugs they can actually work on.

## Helpful GitHub Filters

### Triage Process
1. [Newly created issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Anew+label%3Abug+-label%3Abackend%2Foss+-label%3Abackend%2Fk8s+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3A%22waiting+for+reproduction%22+-label%3A%22waiting-response%22+-label%3Aexplained+) require initial filtering.
2. Clarify [unreproduced issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+created%3A%3E2020-05-01+-label%3Abackend%2Fk8s+-label%3Aprovisioner%2Fsalt-masterless+-label%3Adocumentation+-label%3Aprovider%2Fazuredevops+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent+-label%3Abackend%2Fmanta+-label%3Abackend%2Fatlas+-label%3Abackend%2Fetcdv3+-label%3Abackend%2Fetcdv2+-label%3Aconfirmed+-label%3A%22pending+project%22+-label%3Anew+-label%3A%22waiting+for+reproduction%22+-label%3Awaiting-response+-label%3Aexplained+sort%3Acreated-asc+)
3. Explain or fix [confirmed issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+-label%3Aexplained+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+). Prioritize [confirmed crashes](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Acrash+label%3Abug+-label%3Aexplained+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+).
4. Fix [explained issues](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+label%3Aexplained+no%3Amilestone+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+label%3Aconfirmed+-label%3A%22pending+project%22+)

### Other Backlog

[Confirmed needs for documentation fixes](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+label%3Adocumentation++label%3Aconfirmed+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+)

[Confirmed bugs that will require significant projects to fix](https://github.com/hashicorp/terraform/issues?q=is%3Aopen+label%3Abug+label%3Aconfirmed+label%3A%22pending+project%22+-label%3Abackend%2Fk8s+-label%3Abackend%2Foss+-label%3Abackend%2Fazure+-label%3Abackend%2Fs3+-label%3Abackend%2Fgcs+-label%3Abackend%2Fconsul+-label%3Abackend%2Fartifactory+-label%3Aterraform-cloud+-label%3Abackend%2Fremote+-label%3Abackend%2Fswift+-label%3Abackend%2Fpg+-label%3Abackend%2Ftencent++-label%3Abackend%2Fmanta++-label%3Abackend%2Fatlas++-label%3Abackend%2Fetcdv3++-label%3Abackend%2Fetcdv2+)

### Milestone Use

Milestones ending in .x indicate issues assigned to that milestone are intended to be fixed during that release lifecycle. Milestones ending in .0 indicate issues that will be fixed in that major release. For example:

[0.13.x Milestone](https://github.com/hashicorp/terraform/milestone/17). Issues in this milestone should be considered high-priority but do not block a patch release. All issues in this milestone should be resolved in a 13.x release before the 0.14.0 RC1 ships.

[0.14.0 Milestone](https://github.com/hashicorp/terraform/milestone/18). All issues in this milestone must be fixed before 0.14.0 RC1 ships, and should ideally be fixed before 0.14.0 beta 1 ships.

[0.14.x Milestone](https://github.com/hashicorp/terraform/milestone/20). Issues in this milestone are expected to be addressed at some point in the 0.14.x lifecycle, before 0.15.0. All issues in this milestone should be resolved in a 14.x release before the 0.15.0 RC1 ships.

[0.15.0 Milestone](https://github.com/hashicorp/terraform/milestone/19). All issues in this milestone must be fixed before 0.15.0 RC1 ships, and should ideally be fixed before 0.15.0 beta 1 ships.
