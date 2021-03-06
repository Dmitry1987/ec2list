ec2list

The tool used to quickly search through all EC2 instances and show its public IP.
Easy filtering by words from "Name" tag. Uses ```~/.aws/credentials``` file to access AWS instances information. Caching results for 1 hour, to not query AWS each time (caching file will be named ```cache.gob``` and it saves to same location where the binary is).

After compiling, or downloading the release, put the binary in your PATH (depends on your OS). And make sure you have the ```~/.aws/credentials``` file with at least "ReadOnly" permissions to your AWS account.

Usage:
```ec2list <some keyword> <another keyword>```

Keywords above, are parts of your Name tag that you're looking for, optional. If run with no keywords, it will show table of *all* your EC2 instances and their Public IPs.

Example: 
Lets say we have servers named 
+ ```kubernetes master```
+ ```kubernetes minion staging```
+ ```kubernetes minion production```
(minions are many servers). 

You want to quickly list IPs of all ```kubernetes minion staging``` (by Tag Name of EC2 instance)
- Run: ```ec2list kub mi stag``` it will compare the half words you typed - to words in name tag, and if all 3 are found, it will include that IP to list that will be shown to you.

It simplified searches a lot in our department, we have loooong naming convention on servers, and thousands of ec2 instances.
Searching in AWS web console was less convenient. With this tool you don't need to leave the command line, to find needed server to ssh to.

*Can easily SSH into found servers by chosing number* (list ordered by numbers, and pressing corresponding number opens ssh to that server).
