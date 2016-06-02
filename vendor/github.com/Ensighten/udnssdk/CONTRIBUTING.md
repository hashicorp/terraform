# Contributing

Want to contribute? Up-to-date pointers should be at:
<http://contributing.appspot.com/udnssdk>

Got an idea? Something smell wrong? Cause you pain? Or lost seconds of
your life you'll never get back?

All contributions are welcome: ideas, patches, documentation, bug
reports, complaints, and even something you drew up on a napkin.

Programming is not a required skill. Whatever you've seen about open
source and maintainers or community members saying "send patches or die":
you will not see that here.

It is more important to me that you are able to contribute. If you
haven't got time to do anything else, just email me and I'll try to
help: <joseph@josephholsten.com>.

I promise to help guide this project with these principles:

-   Community: If a newbie has a bad time, it's a bug.
-   Software: Make it work, then make it right, then make it fast.
-   Technology: If it doesn't do a thing today, we can make it do
    it tomorrow.

Here are some ways you can be part of the community:

## Something not working? Found a Bug?

Find something that doesn't feel quite right? Here are 5 steps to
getting it fixed!

### Check your version

To make sure you're not wasting your time, you should be using the
latest version before you file your bug. First of all, you should
download the latest revision to be sure you are up to date. If you've
done this and you still experience the bug, go ahead to the next step.

### Search our [issues]

Now that you have the latest version and still think you've found a bug,
search through issues first to see if anyone else has already filed it.
This step is very important! If you find that someone has filed your bug
already, please go to the next step anyway, but instead of filing a new
bug, comment on the one you've found. If you can't find your bug in
issues, go to the next step.

### Create a Github account https://github.com/join

You will need to create a Github account to be able to report bugs (and
to comment on them). If you have registered, proceed to the next step.

### File the bug!

Now you are ready to file a bug. The [Writing a Good Bug Report]
document gives some tips about the most useful information to include in
bug reports. The better your bug report, the higher the chance that your
bug will be addressed (and possibly fixed) quickly!

### What happens next?

Once your bug is filed, you will receive email when it is updated at
each stage in the bug life cycle. After the bug is considered fixed, you
may be asked to download the latest revision and confirm that the fix
works for you.

## Submitting patches

1.  [Fork the repository.]
2.  [Create a topic branch.]
3.  Add specs for your unimplemented feature or bug fix.
4.  Run `script/test`. If your specs pass, return to step 3.
5.  Implement your feature or bug fix.
6.  Run `script/test`. If your specs fail, return to step 5.
7.  Add, commit (say *why* the changes were made, we can look at the
    diff to see *how* they were made.), and push your changes. For
    documentation-only fixes, please add `[ci skip]` to your commit
    message to avoid needless CI builds.
8.  [Submit a patch.]

## Setting up a local dev environment

For those of you who do want to contribute with code, we've tried to
make it easy to get started. You can install all dependencies and tools
with:

    script/bootstrap

Good luck!

## Style guide

There are great style guides out there, we don't need to reinvent the
wheel. Here are ones we like:

-   `go`: https://code.google.com/p/go-wiki/wiki/CodeReviewComments
-   `sh`: http://google.github.io/styleguide/shell.xml
-   `ruby`: https://github.com/bbatsov/ruby-style-guide
-   `python`: https://www.python.org/dev/peps/pep-0008/

For some things, the best we've got is a decent formatting tool:

-   `markdown`: `pandoc --to=markdown --reference-links --atx-headers --columns 72`
-   `json`: `jq .`

  [issues]: https://github.com/Ensighten/udnssdk/issues
  [Writing a Good Bug Report]: http://www.webkit.org/quality/bugwriting.html
  [Fork the repository.]: https://help.github.com/articles/fork-a-repo
  [Create a topic branch.]: http://learn.github.com/p/branching.html
  [Submit a patch.]: https://help.github.com/articles/using-pull-requests
