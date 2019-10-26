# Differences between Dsiem and OSSIM

Aside from the obvious things like UI, listed features, or storage backend, there are also differences between Dsiem and OSSIM that may not be apparent from the start. This page aims to explain such things to the more experienced OSSIM users.

## Handling vulnerability information

OSSIM maintains a table that maps CVE numbers to Suricata plugin SIDs. When an event matches one of those SIDs, OSSIM will search its vulnerability database for the IP address and port from the event, and the corresponding CVE number from the mapping table. If the search returns a positive result, OSSIM will raise the event reliability score to 10 (max.), which in turn will raise the associated alarm's risk value.

OSSIM can do the above, because deep integration with Suricata and OpenVAS *is* part of its scope. Dsiem can't do it because avoiding dependency with specific tools is part of our design goal. We (or anyone else) may build such integration into Dsiem for a particular deployment, but those changes likely will not be part of the main Dsiem code base.

The downside is, Dsiem can only search vulnerability information sources for an IP and port pair of an event, without a way of verifying whether any of the vulnerability found is indeed relevant to the event. Consequently, positive vulnerability search result in Dsiem is only used to decorate alarms in the same way that threat intelligence information is, i.e. no risk (re)assessment will be made based on their results. So it is up to the analyst to assess the relevance of those listed vulnerabilities to the associated alarm.

## Correlation rules structure

Correlation rules in OSSIM are modelled like a tree, allowing directive to have multiple paths towards the last stage. From the [documentation](https://www.alienvault.com/documentation/usm-appliance/correlation/about-correlation-rules.htm):

> When a correlation directive contains multiple rules, the indentation of the rules reveals the relationship between the rules. Indented rules have an AND relationship while parallel rules have an OR relationship.

In contrast, Dsiem correlation rules are structured like a list that can only progress in one path. This is because in our experience, most use cases are much better off expressed as multiple directives instead of a single directive with many branches. Multiple directives produce more concise alarm's title, and users are less likely to make logical mistakes in formulating them.

There is one common and valid use of multiple path in OSSIM rules though, and that is to capture extra events from the previously completed stages. Typically, this is done to prevent those events from creating another instance of the correlation rules tracker (or backlog in OSSIM/Dsiem parlance) while there's one still active in memory. To support this useful behaviour, Dsiem directive has an extra flag `all_rules_always_active` that can be activated to achieve the same result.

For instance, suppose we want to detect SSH bruteforce attempt, where bruteforce is defined as *5 or more failed logins, followed by a successful login within 5 minutes time frame*. The OSSIM rules for this look something like:

```
level 1: SSH failed login (occurrence: 1)
  level 2: SSH failed login (occurrence: 4, timeout: 300s)
    level 3a: SSH succesful login (occurrence: 1, timeout: 300s)
    level 3b: SSH Failed login (occurrence: 1000, timeout: 300s)
```
The above directive uses rule 3a to implement the main goal, and 3b to capture extra failed login events that may occur before rule 3a's condition is satisfied.

The above rules can be implemented in Dsiem as follows:

```
"all_rules_always_active": true
level 1: SSH failed login (occurrence: 1)
level 2: SSH failed login (occurrence: 4, timeout=300s)
level 3: SSH succesful login (occurrence: 1, timeout=300s)
```
When the above directive processing reaches level 3, `all_rules_always_active: true` will cause it to keep evaluating new events against level 1 and level 2 rules, thereby preventing subsequent SSH failed login events from creating a new tracker (backlog) instance for the same directive.