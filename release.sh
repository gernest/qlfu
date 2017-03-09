for f in *;do
  if [[ -d $f ]]; then
  echo $f.tar.gz
  tar -zcvf $f.tar.gz  $f
  fi
done
  