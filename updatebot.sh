#!/bin/bash
jx step create pr go --name github.com/jenkins-x/lighthouse --version $VERSION --build "make mod" --repo https://github.com/cloudbees/lighthouse-githubapp.git
