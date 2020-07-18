#### Minimal Google cloud storage cp

when you want to use google cloud storage, but don't have python

#### Usage

```sh
gsutil cp <src> <dst>
```

#### To bucket

```sh
echo "hi" | gsutil cp - gs://bucket/key

gsutil cp ./key gs://bucket/key
```

#### From bucket

```sh
gsutil cp gs://bucket/key .

gsutil cp gs://bucket/key -

gsutil cp gs://bucket/key ./key.txt
```
