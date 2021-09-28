
checkcount=10
tempcount=0

while true; do
  ready=$(oc get deploy nfs-provisioner --no-headers|awk '{print $2}'|cut -d/ -f1)
  desired=$(oc get deploy nfs-provisioner --no-headers|awk '{print $2}'|cut -d/ -f2)

  if [[ $ready == $desired ]]
  then
    echo "NFS is Ready!"
    break
  else 
    tempcount=$((tempcount+1))
    echo "NFS is not Ready: $tempcount times"
    echo "Wait for 10 secs"

    sleep 10
  fi
  if [[ $ready != $desired ]] && [[ $checkcount == $tempcount ]]
  then
    echo "1"
    break
  fi
done