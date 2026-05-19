Git Merge driver for json files
===========================

Why
---

Git raises merge conflict when changes happen on [adjacent lines](https://softwareengineering.stackexchange.com/a/311786)  
While helpful for a source code, it is nonsensical for structured data files like json, where changes to different keys happen often and are isolated by definition

Existing tools either don't do deep merge, or fail to produce expected output. And bringing whole node.js/deno into a cicd worker for a ~100 line js script is excessive

Limitations
-----------

Key order is not preserved  
Arrays could not be merged

How to use
----------

Put binary to a known location, preferably in PATH  

Run in you repo before merge:
```
git config merge.json.driver "json-merge-driver %O %A %B"
echo '*.json merge=json' >> .gitattributes
```

License
-------

Licensed under Apache License version 2.0. See the [`LICENSE`](LICENSE) file for
details.
