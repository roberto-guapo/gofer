#!/usr/bin/env bash
# Rebuilds pi as upstream's master plus a merge of each branch listed in
# fork/branches.txt on the fork/tools branch. See FORK-MAINTENANCE.md.
#
# Usage: fork/rebuild-pi.sh [--no-fetch] [--target BRANCH]
#
#   --no-fetch       build from the upstream/master already fetched
#   --target BRANCH  build BRANCH instead of pi, to try a rebuild out
#
# The new branch is built in a separate worktree, and the checkout this runs
# from is only moved at the end, once everything merged and built. The old
# branch is kept as backup/<branch>-before-rebuild-<time>. Nothing is pushed.
set -euo pipefail

target=pi
fetch=1
while [ $# -gt 0 ]; do
	case $1 in
	--no-fetch) fetch=0 ;;
	--target)
		target=${2:?--target needs a branch name}
		shift
		;;
	-h | --help)
		sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "rebuild-pi: unknown option $1 (see --help)" >&2
		exit 2
		;;
	esac
	shift
done

die() {
	echo "rebuild-pi: $*" >&2
	exit 1
}

base=upstream/master
tools=fork/tools
list=fork/branches.txt

root=$(git rev-parse --show-toplevel) || die "run this inside the Gofer repository"
cd "$root"
work="$(dirname "$root")/$(basename "$root")-$target-rebuild"

# Conflict resolutions are recorded once and replayed by later rebuilds.
git config rerere.enabled true
git config rerere.autoupdate true

if [ "$fetch" = 1 ]; then
	echo "Fetching upstream..."
	git fetch --quiet upstream
fi

# The list is read from the fork/tools branch, not from this checkout, so
# every rebuild uses the committed list whatever is checked out.
branches=$(git show "$tools:$list" | sed -e 's/#.*//' -e 's/[[:space:]]*$//' -e '/^$/d') ||
	die "can't read $list from the $tools branch"
[ -n "$branches" ] || die "$list lists no branches"
for b in $branches; do
	git rev-parse --verify --quiet "refs/heads/$b" >/dev/null || die "$b, listed in $list, isn't a branch here"
	if [ "$(git rev-list --count "$base..refs/heads/$b")" = 0 ]; then
		echo "Note: $b has nothing that $base doesn't; it can come off the list."
	fi
done

templ_wanted=$(go list -m -f '{{.Version}}' github.com/a-h/templ)
templ_found=$(templ version 2>/dev/null || true)
[ "$templ_found" = "$templ_wanted" ] ||
	die "templ $templ_wanted is needed to regenerate templates (found: ${templ_found:-none})"

# A worktree left by a run that stopped on a conflict is replaced. A
# resolution committed in it has already been recorded.
if [ -e "$work" ]; then
	git worktree remove --force "$work" 2>/dev/null || die "$work is in the way; move it and try again"
fi
git worktree add --quiet --detach "$work" "$base"
cd "$work"

for b in $branches; do
	echo "Merging $b..."
	if git merge --quiet --no-ff --no-edit -m "Merge $b into $target" "refs/heads/$b" >/dev/null 2>&1; then
		continue
	fi
	# Whatever rerere resolved is already staged. Generated templates are
	# taken from either side here and rebuilt from the merged .templ files
	# below, so only hand-written files need a person.
	unresolved=$(git diff --name-only --diff-filter=U)
	by_hand=$(printf '%s\n' "$unresolved" | grep -v '_templ\.go$' || true)
	if [ -n "$by_hand" ]; then
		{
			echo "rebuild-pi: $b conflicts with what was merged before it, in:"
			printf '  %s\n' $by_hand
			echo "Resolve them in $work, commit there with \"git commit --no-edit\""
			echo "so Git records the resolution, and run this script again."
			echo "$target hasn't been changed."
		} >&2
		exit 1
	fi
	for f in $unresolved; do
		git checkout --quiet --ours -- "$f"
	done
	git add -A
	git commit --quiet --no-edit
done

# Text merges of generated files can be out of step with their templates even
# without a conflict, so they're always rebuilt.
echo "Regenerating templates..."
templ generate --ignore-pattern='(^|/)dist(/|$)' >/dev/null 2>&1 ||
	die "templ generate failed in $work; $target hasn't been changed"
if [ -n "$(git status --porcelain)" ]; then
	git add -A
	git commit --quiet -m "Regenerate templates after merging the fork's branches"
fi

echo "Building..."
go build ./... || die "the merged code doesn't build; it's in $work, and $target hasn't been changed"

new=$(git rev-parse HEAD)
cd "$root"
backup=
if old=$(git rev-parse --verify --quiet "refs/heads/$target"); then
	backup="backup/$target-before-rebuild-$(date +%Y%m%d-%H%M%S)"
	git branch "$backup" "$old"
fi
# A checkout that has the branch is moved with it. --keep refuses, and changes
# nothing, if that would overwrite uncommitted changes there.
checkout=$(git worktree list --porcelain | awk -v ref="branch refs/heads/$target" '/^worktree /{w=substr($0,10)} $0==ref{print w}')
if [ -n "$checkout" ]; then
	git -C "$checkout" reset --quiet --keep "$new" ||
		die "$checkout has uncommitted changes in the way; commit or stash them, then: git -C \"$checkout\" reset --keep $new"
else
	git branch --force "$target" "$new"
fi
git worktree remove --force "$work"

echo
echo "$target is now $base ($(git log --format=%h -1 "$base")) with:"
for b in $branches; do
	n=$(git rev-list --count "$base..refs/heads/$b")
	if [ "$n" = 1 ]; then s=; else s=s; fi
	printf '  %s (%s commit%s)\n' "$b" "$n" "$s"
done
[ -z "$backup" ] || echo "The previous $target is kept as $backup."
echo "Nothing was pushed. Test it, then publish it with:"
echo "  git push --force-with-lease origin $target"
