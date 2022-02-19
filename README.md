# starhook [![](https://github.com/fatih/starhook/workflows/build/badge.svg)](https://github.com/fatih/starhook/actions)

Manage & Analyze repositories at scale

# Install

```bash
go get github.com/fatih/starhook/cmd/starhook
```


# Usage


### Initialize & clone repositories

First, let us initialize starhook to sync the repositories. A set of repositories is called a `reposet` and you can have multiple reposets based on different queries or even GitHub tokens. Let's start adding our first `reposet`.

Pass the GitHub token, the location to store your repositories and the query needed to fetch the repositories:

```
$ mkdir -p /path/to/repos
$ starhook config add --token=$GITHUB_TOKEN --dir /path/to/repos --query "user:fatih language:go" 
starhook is initialized (config name: 'wonderful-star')

Please run 'starhook sync' to download and sync you repositories.
```

Now, let's clone the repositories  with the `--dry-run` flag to see what is `starhook` planning to do:

```
$ starhook sync --dry-run
querying for latest repositories ...
last synced: a long while ago
updates found:
  clone  :  29
  update :   0

remove -dry-run to update & clone the repositories
```

As you see, it found `29` repositories. This command will all repositories that
belongs to user `fatih` and are classified as `go` programming language. For
more information about the `query` parameter, checkout
https://docs.github.com/en/search-github/getting-started-with-searching-on-github/about-searching-on-github. 

Now, let's remove the `--dry-run` flag. `starhook` will execute the query and clone the repositories: 

```
$ starhook sync
querying for latest repositories ...
last synced: a long while ago
updates found:
  clone  :  29
  update :   0
  ...
  ...
cloned: 29 repositories (elapsed time: 10.146454763s)
```

### Update repositories

To update existing repositories, just run the `sync` subcommand. `starhook` only updates repositores that have new changes:


```
$ starhook sync
querying for latest repositories ...
last synced: 20 minutes ago
updates found:
  clone  :   0
  update :   1
  "starhook" is updated (last updated: 20 minutes ago)
updated: 1 repositories (elapsed time: 2.032119469s)
```


### List repositories

To list all existing repositories, use the `list` subcommand and run the following command:

```
$ starhook list
  1 fatih/pool
  2 fatih/set
  3 fatih/structs
  4 fatih/color
  5 fatih/gomodifytags
  ...
local 29 repositories (last synced: 15 minutes ago)
```


### Delete repositories

To delete a repository from the local storage, use the `delete` subcommand with the `--id` flag:

```
$ starhook delete --id 3
removed repository: "fatih/structs"
```

### Create a second reposet

As we said earlier, we can manage multiple `reposet`'s. Let's create another reposet, but this time for repositories that are written in VimScript:

```
$ mkdir -p /path/to/viml-repos
$ starhook config add --token=$GITHUB_TOKEN --dir /path/to/viml-repos --query "user:fatih language:viml" 
starhook is initialized (config name: 'shining-moon')

Please run 'starhook config switch shining-moon && starhook sync' to download and sync you repositories.
```

Let's see all current reposets:

```
$ starhook config list
Name                     wonderful-star
Query                    user:fatih language:go
Repositories Directory   /path/to/repos

Name                     shining-moon
Query                    user:fatih language:viml
Repositories Directory   /path/to/viml-repos
```

Let's show the current selected reposet configuration:


```
$ starhook config show
Name                     wonderful-star
Query                    user:fatih language:go
Repositories Directory   /path/to/repos
```


To use the new reposet, we need to switch and sync the new `reposet`:

```
$ starhook config switch shining-moon
Switched to 'shining-moon'

$ starhook sync
querying for latest repositories ...
last synced: a long while ago
updates found:
  clone  :   5
  update :   0
  cloning vim-hclfmt
  cloning vim-go-tutorial
  cloning vim-nginx
  cloning vim-go
  cloning dotfiles
  "vim-go" is created
  "dotfiles" is created
  "vim-hclfmt" is created
  "vim-go-tutorial" is created
  "vim-nginx" is created
cloned: 5 repositories (elapsed time: 2.279053145s)
```

