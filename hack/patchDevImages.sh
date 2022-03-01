#!/usr/bin/env bash
#
# Patch Lighthouse images in cluster with the latest development images
#
# Prerequisites:
# - connection to the cluster
# - ability to push images to a repository which can be accessed by the cluster

programname=$0
#######################################
# Writes error message to STDERR
#######################################
err() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $*" >&2
}

#######################################
# Writes usage message
#######################################
function usage {
    echo "usage: $programname [-r repository] [-u user] [-t tag] [-n namespace]"
    echo "  -r     The repository host and port to push the images to. Defaults to empty string, aka Docker Hub"
    echo "  -u     The user of the repository. Defaults to empty."
    echo "  -t     The tag to give the image. Defaults to 'latest'"
    echo "  -n     The namespace in which Lighthouse is installed. Defaults to 'lighthouse'."
    exit 1
}

# Default the registry and user of the image to the empty string, image tag to 'latest' and default deploy namespace to 'lighthouse'
registry=""
user=""
tag="latest"
namespace="lighthouse"

while getopts ":r:u:t:n:" opt; do
  case ${opt} in
    r )
      registry=$OPTARG
      ;;
    u )
      user=$OPTARG
      ;;
    t )
      tag=$OPTARG
      ;;
    n )
      namespace=$OPTARG
      ;;
    \? )
      usage
      exit 1
      ;;
    : )
      usage
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))

if ! make build-linux; then
  err "Unable to generate binaries"
  exit 1
fi

for val in {0..5}; do
    docker_file=(webhooks keeper foghorn gc tekton jenkins)
    image_names=(webhooks keeper foghorn gc-jobs tekton-controller jenkins-controller)

    full_image_name=""
    if [ -n "${registry}" ]
     then full_image_name="${full_image_name}${registry}/"
    fi
    if [ -n "${user}" ]
     then full_image_name="${full_image_name}${user}/"
    fi
    full_image_name="${full_image_name}lighthouse-${image_names[$val]}:$tag"

    if ! docker build -f ./docker/${docker_file[$val]}/Dockerfile -t "${full_image_name}" .; then
      err "Unable to build images for ${image_names[$val]}"
      exit 1
    fi

    if !  docker push "${full_image_name}"; then
      err "Unable to push images for ${image_names[$val]}"
      exit 1
    fi

    if  kubectl -n "${namespace}" get deployment lighthouse-"${image_names[$val]}" 2>/dev/null; then
      kubectl -n "${namespace}" patch deployment lighthouse-"${image_names[$val]}" -p "$(cat <<EOF
spec:
  template:
    metadata:
      labels:
        redeploy: "$(date +%s)"
    spec:
      containers:
      - name: lighthouse-${image_names[$val]}
        image: ${full_image_name}
        imagePullPolicy: Always

EOF
)"
   fi

done;
