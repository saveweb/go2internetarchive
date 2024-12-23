# WIP - Yet another implementation of the IA S3-like API Client in Go


## Metadata

### Metadata Key

- [x] Handle all kinds of illegal characters and edge cases!

### Metadata Value

Valid XML characters.

- [x] Rewrite my previous XML illegal characters filter in Go. (saveweb/biliarchiver/utils/xml_chars.py) 

## Uploading

- [ ] Queue derive
- [ ] Checksum
- [x] PUT
- [ ] Multipart Upload
   - [ ] Resumable Upload
- [ ] DELETE
- [x] Implement the `upload` function.

## Misc

- [ ] ini parser (.config/internerarchive/ia.ini)
- [ ] Multi Account Support
- [ ] Downloade
- [ ] Uploade
- [ ] Metadata
- [ ] Search
- [ ] Task