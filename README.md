# starhook [![](https://github.com/fatih/starhook/workflows/build/badge.svg)](https://github.com/fatih/starhook/actions)

Search & Analyze repositories in scale

# Install

```bash
go get github.com/fatih/starhook/cmd/starhook
```


# Usage


### Initialize & clone repositories

Initialize and sync the repositories. Let's run first with the `--dry-run` flag to see what is `starhook` planning to do:

```
$ starhook --token=$GITHUB_TOKEN --dir /path/to/repos --query "user:fatih language:go" --dry-run --sync
==> querying for latest repositories ...
==> last synced: a long while ago
==> updates found:
  clone  :  29
  update :   0

remove -dry-run to update & clone the repositories
```

This command will all repositories that belongs to user `fatih`
and are classified as `go` programming language. For more information about the `query` parameter, checkout https://docs.github.com/en/search-github/getting-started-with-searching-on-github/about-searching-on-github. As you see, it found `29` repositories.


Now, let's remove the `--dry-run` flag. `starhook` will execute the query and clone the repositories: 

```
$ starhook --token=$GITHUB_TOKEN --dir /path/to/repos --query "user:fatih language:go" --sync
==> querying for latest repositories ...
==> last synced: a long while ago
==> updates found:
  clone  :  29
  update :   0
  ...
  ...
==> cloned: 29 repositories (elapsed time: 10.146454763s)
```

### Update repositories

To update existing repositories, just run the same command. `starhook` only updates repositores that have new changes. You don't need to pass the `--query` flag anymore as it's saved.:


```
$ starhook --token=$GITHUB_TOKEN --dir /path/to/repos --sync
==> querying for latest repositories ...
==> last synced: 20 minutes ago
==> updates found:
  clone  :   0
  update :   1
  "starhook" is updated (last updated: 20 minutes ago)
==> updated: 1 repositories (elapsed time: 2.032119469s)
```


### List repositories

To list all existing repositories, use the `--list` flag and run the following command:

```
$ starhook --token=$GITHUB_TOKEN --dir /path/to/repos --list
  1 fatih/pool
  2 fatih/set
  3 fatih/structs
  4 fatih/color
  5 fatih/gomodifytags
  ...
==> local 29 repositories (last synced: 15 minutes ago)
```


### Delete repositories

To delete a repository from the local storage, pass the `id` from the list:

```
$ starhook --token=$GITHUB_TOKEN --dir /path/to/repos --delete 3
==> removed repository: "fatih/structs"
```
