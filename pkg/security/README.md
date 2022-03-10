Lighthouse Pipeline Security
============================

Adds support for security policies that are matching pipelines by `repository-owner/repository-name` using regexp patterns.

Introduced with https://github.com/jenkins-x/lighthouse/pull/1424

**Allows enforcements of:**
- namespace
- service account
- job maximum execution time limit


Concept
-------

Policies are stored in central namespace e.g. in `jx`. Every time a job have to be spawned a PipelineSecurityPolicy is checked. If it matches, then enforcements are applied to `LighthouseJob` and to `PipelineRun`.


Relation with Tekton's PipelineRun
----------------------------------

First `LighthouseJob` is created, then a Tekton's `PipelineRun` is made out of it. 

Both `LighthouseJob` and `PipelineRun` are having a label `lighthouse.jenkins-x.io/securityPolicyName` containing a policy name that was matched in time of `LighthouseJob` creation.


Edge cases
----------

- There are policies, but regexp is not compillable. Then do not schedule any jobs. Because we want security policies, but they are invalid, we do not know what to allow
- Multiple policies matched for a Job. Then do not start this job, security must be defined to be unambiguous and simple
