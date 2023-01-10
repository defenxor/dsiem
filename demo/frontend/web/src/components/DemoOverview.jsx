import React, { useEffect, useCallback } from 'react'
import mermaid from 'mermaid'
import { windowNavigate } from './utils.js'
import PropTypes from 'prop-types'
import {
  EuiEmptyPrompt,
  EuiButton,
  EuiPageBody,
  EuiPageSection,
  EuiFlexGroup,
  EuiFlexItem
} from '@elastic/eui'

mermaid.initialize({
  startOnLoad: true,
  flowchart: {
    useMaxWidth: false
  }
})

export const DemoOverview = props => {
  const { targetUrl = `http://${window.location.hostname}:8081` } = props
  const {
    chart = `
        graph TD
        A("Attacker (you)")
        B[Switch]
        C(Suricata NIDS)
        D(Target Apache Web Server)
        E(Ossec HIDS)
        F(Auditbeat)
        G(Logstash)
        H(Elasticsearch)
        I(Dsiem)
        J(Filebeat)
        A -->|Shellshock exploit / HTTP| B
        D --- |detect file changes / OS hook| E
        D --- |detect file changes / OS hook| F
        B --> |Shellshock exploit / HTTP| D
        B --> |packet trace| C
        C --> |eve JSON log / beats| G
        E --> |file integrity log / syslog| G
        G --> |normalized events & alarms / HTTP| H
        F --> |file integrity log / beats| H
        G --> |normalized events / HTTP|I
        I --> |alarms / JSON file| J
        J --> |alarms / beats | G
      `
  } = props

  useEffect(
    () => {
      mermaid.contentLoaded()
    },
    [chart]
  )

  const openTarget = useCallback(
    () => {
      windowNavigate(targetUrl)
    },
    [targetUrl]
  )

  return (
    <EuiPageBody>
      <EuiPageSection>
        <EuiEmptyPrompt
          iconType='questionInCircle'
          title={<h1>Demo Guide</h1>}
          body={
            <>
              <p>
                Review the logical network setup below. Open the target web page
                once you&lsquo;re ready to proceed.
              </p>
              <p>
                After that, once Kibana is ready, you can begin using the cards
                above. Start by exploiting the target multiple times to generate
                alarm.
              </p>
            </>
          }
          actions={
            <EuiButton
              color='primary'
              // fill
              onClick={openTarget}
              iconType='bullseye'
            >
              Open the target web page
            </EuiButton>
          }
        />
        <EuiPageSection>
          <EuiFlexGroup
            // style={divStyle}
            justifyContent='spaceAround'
            alignItems='center'
          >
            <EuiFlexItem grow={false}>
              <div className='mermaid'>{chart}</div>
            </EuiFlexItem>
          </EuiFlexGroup>
        </EuiPageSection>
      </EuiPageSection>
    </EuiPageBody>
  )
}

DemoOverview.propTypes = {
  targetUrl: PropTypes.string,
  chart: PropTypes.string
}
