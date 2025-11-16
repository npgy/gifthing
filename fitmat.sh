if [ -z "$1" ]; then
  echo "Please specify an input file"
  return 1 2>/dev/null
  exit 1
elif [ -z "$2" ]; then
  echo "Please specify an output file"
  return 1 2>/dev/null
  exit 1
fi

WIDTH=920
HEIGHT=600

ffmpeg -i $1 -vf "scale=$WIDTH:$HEIGHT:force_original_aspect_ratio=decrease,pad=$WIDTH:$HEIGHT:-1:-1:color=black" $2