#!/bin/sh
set -ue

#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'

if [ "$#" -lt 1 ]
then
	printf "Usage:\n\t%s a.go ...\n\t%s -all\n\t%s -u\n" "$0" "$0" "$0" >&2
	exit 1
fi

UPDATE=0
SIMPLIFY=0
FMTALL=0

while [ $# -gt 0 ]
do
	x="$1"
	case "$x" in
		-u)
			shift
			UPDATE=1
			;;
		-s)
			shift
			SIMPLIFY=1
			;;
		-a|-all)
			shift
			FMTALL=1
			;;
		-*)
			echo "Unknown option: $x" 1>&2
			exit 1
			;;
		*)
			break
			;;
	esac
done

if [ $UPDATE = 1 ]
then
	cd # otherwise it'll add dep to current go.mod
	go get -u golang.org/x/tools/cmd/goimports
	echo "Updated." >&2
	exit
fi

export GOPATH=`go env GOPATH`:`pwd`

dofmtsimplify ()
{
	find "$1" \
		-type f \
		-name '*.go' \
		-not -regex ".*_nofmt[._].*" \
		-exec \
			gofmt -s -w '{}' ';'
}

dofmtimports ()
{
	find "$1" \
		-type f \
		-name '*.go' \
		-not -regex ".*_nofmt[._].*" \
		-exec \
			goimports -local "$2/" -w '{}' ';'
}

dofmtone ()
{
	printf 'formatting [%s] [%s]\n' "$1" "$2"

	if [ $SIMPLIFY = 1 ]
	then
		dofmtsimplify "$1"
	fi
	dofmtimports "$1" "$2"
}

dofmt ()
{
	if [ "$1" = 'src' -o "$1" = 'src/' ]
	then
		dofmt src/*
		return
	fi

	for x in "$@"
	do
		case "$x" in
		src/*)
			# ok
			;;
		*)
			echo "Bad fmt path: $x" >&2
			exit 1
			;;
		esac

		m=${x#src/} # strip src prefix
		m=${m%%/*}  # strip anything other than module name

		if [ "$m" = '' ]
		then
			echo "Bad fmt path: $x" >&2
			exit 1
		fi

		dofmtone "$x" "$m"
	done
}

if [ $FMTALL = 1 -o $# -lt 1 ]
then
	cd $(dirname "$0")
	dofmt src
	exit
fi

dofmt "$@"
