import React, { useState, useEffect, useCallback } from 'react'
import { Iframe } from './Iframe.jsx'
import { isChrome, untilKibanaIsReady, shellshockSend } from './utils.js'
import { Card } from './Card.jsx'
import {
  EuiSpacer,
  EuiFlexGroup,
  EuiFlexItem,
  EuiPageBody,
  EuiPageContent,
  EuiPageContentBody,
  EuiGlobalToastList,
  EuiAccordion,
  EuiHeader,
  EuiHeaderSectionItem,
  EuiHeaderLogo,
  EuiHeaderLinks,
  EuiHeaderLink,
  EuiLoadingContent,
  EuiButtonToggle
} from '@elastic/eui'

export const DemoMenu = props => {
  const baseUrl = `${window.location.protocol}//${window.location.hostname}`
  const appPort = window.location.port
  const targetHost = `${baseUrl}:8081`
  const targetUrl = `${targetHost}/cgi-bin/vulnerable`
  const dsiemUrl = `${baseUrl}:8080/ui/`
  const apmUrl = `${baseUrl}:5601/app/apm#/services/dsiem`
  const kibanaDashboard =
    'app/kibana#/dashboard/87c18520-b337-11e8-b3e4-11404c6637fe'
  const kibanaUrl = `${baseUrl}:5601/${kibanaDashboard}`
  const kibanaCheckUrl = `${baseUrl}:${appPort}/kibana/${kibanaDashboard}`
  const elasticSIEMUrl = `${baseUrl}:5601/app/siem#/hosts/events?kqlQuery=(filterQuery:(expression:'threat.framework%20:*',kind:kuery),queryLocation:hosts.page)`
  const directiveUrl = `${baseUrl}:${appPort}/#/directive`
  const overviewUrl = `${baseUrl}:${appPort}/#/overview`
  const docsUrl =
    'https://github.com/defenxor/dsiem/tree/master/docs#dsiem-documentation'
  const codeUrl = 'https://github.com/defenxor/dsiem/'
  const helpUrl = 'https://github.com/defenxor/dsiem/issues'

  const [toasts, setToasts] = useState([])
  const [iframeUrl, setIframeUrl] = useState(overviewUrl)
  const [loading, setLoading] = useState(true)
  const [useTab, setTab] = useState(false)
  const [toastId, setToastId] = useState(0)
  const [exploitCount, setExploitCount] = useState(0)
  const [exploitOngoing, setExploitOngoing] = useState(false)

  useEffect(
    () => {
      untilKibanaIsReady(kibanaCheckUrl, 3000).then(() => setLoading(false))
    },
    [kibanaCheckUrl]
  )

  const removeToast = useCallback(
    removedToast => {
      setToasts(toasts.filter(toast => toast.id !== removedToast.id))
    },
    [toasts]
  )

  const handleToggleTab = useCallback(e => {
    setTab(e.target.checked)
  }, [])

  const openUrl = useCallback(
    (url, forceTab) => {
      console.log('rerendering?')
      if (useTab || forceTab) {
        window.open(url, '_blank')
      } else {
        setIframeUrl(url)
      }
    },
    [useTab]
  )

  const exploit = useCallback(
    async () => {
      setExploitOngoing(true)
      const r = Math.round(Math.random() * 1000)
      const fileContent =
        '<html><body><h1>Defaced - By #' + r + '</h1></body></html>'
      const file = '/var/www/html/index.html'
      const res = await shellshockSend(targetUrl, file, fileContent)
      let toast
      setToastId(toastId + 1)
      if (res.success) {
        const cnt = exploitCount + 1
        setExploitCount(cnt)
        toast = {
          id: String(toastId),
          title: 'Successful exploitation ' + cnt + 'x',
          color: 'success'
        }
      } else {
        toast = {
          id: String(toastId),
          title: 'Failed exploitation!',
          color: 'danger',
          text: 'status: ' + res.errMsg
        }
      }
      setToasts(toasts.concat(toast))
      if (res.success) {
        openUrl(targetHost + '?' + r)
      }
      setExploitOngoing(false)
    },
    [exploitCount, openUrl, targetHost, targetUrl, toasts, toastId]
  )

  const openDoc = useCallback(
    () => {
      openUrl(docsUrl, true)
    },
    [openUrl, docsUrl]
  )

  const openCode = useCallback(
    () => {
      openUrl(codeUrl, true)
    },
    [openUrl, codeUrl]
  )

  const openHelp = useCallback(
    () => {
      openUrl(helpUrl, true)
    },
    [openUrl, helpUrl]
  )

  return (
    <EuiPageBody>
      <EuiPageContent>
        <EuiPageContentBody>
          <EuiHeader>
            <EuiHeaderSectionItem border='right'>
              <EuiHeaderLogo iconType='securityApp'>Dsiem Demo</EuiHeaderLogo>
            </EuiHeaderSectionItem>
            <EuiHeaderLinks>
              <EuiHeaderLink onClick={openDoc}>Docs</EuiHeaderLink>
              <EuiHeaderLink onClick={openCode}>Code</EuiHeaderLink>
              <EuiHeaderLink iconType='help' onClick={openHelp}>
                Help
              </EuiHeaderLink>
            </EuiHeaderLinks>
          </EuiHeader>
          <EuiSpacer />
          {loading && <EuiLoadingContent lines={1} />}

          <EuiAccordion
            id='acc1'
            buttonContent='Show or hide the menu cards.'
            initialIsOpen
            extraAction={
              <EuiButtonToggle
                isEmpty
                label='open on a new tab'
                iconType={useTab ? 'check' : ''}
                onChange={handleToggleTab}
                isSelected={useTab}
              />
            }
          >
            <EuiSpacer />
            <EuiFlexGroup gutterSize='l' wrap>
              <Card
                logo='graphApp'
                title='Exploit target'
                disabled={loading || exploitOngoing}
                clickHandler={exploit}
                url={targetHost}
                desc={'Shellshock vulnerability @ ' + targetUrl}
              />
              <Card
                logo='logoKibana'
                title='Kibana dashboard'
                disabled={loading}
                clickHandler={openUrl}
                url={kibanaUrl}
                desc='The main analytic UI. Linked to Dsiem UI for alarm management.'
              />
              <Card
                logo='logoWebhook'
                title='Dsiem UI'
                disabled={loading}
                clickHandler={openUrl}
                url={dsiemUrl}
                desc='Manage alarms status/tag, see threat intel/vuln. query results, and pivot to relevant Kibana indices.'
              />
              <Card
                logo='dataVisualizer'
                title='Dsiem directive'
                disabled={loading}
                clickHandler={openUrl}
                url={directiveUrl}
                desc='Review the example directive used on this demo.'
              />
              <Card
                logo='logoSecurity'
                title='Elastic SIEM'
                disabled={loading}
                clickHandler={openUrl}
                url={elasticSIEMUrl}
                desc='Correlate Dsiem alarms further with ECS-compliant events from the Beats family and their modules.'
              />
              <Card
                logo='logoAPM'
                title='Elastic APM'
                disabled={loading}
                clickHandler={openUrl}
                url={apmUrl}
                desc='Dsiem APM integration for performance monitoring and analysis.'
              />
              {isChrome() && (
                <Card
                  logo='addDataApp'
                  title='Open terminal'
                  disabled={loading}
                  clickHandler={openUrl}
                  url='chrome-extension://pnhechapfaindjhompbnflcldabbghjo/html/nassh.html'
                  //
                  desc='Use dpluger to integrate new logs and create Dsiem correlation directives (requires Chrome Secure Shell app).'
                  footerUrl='https://chrome.google.com/webstore/detail/secure-shell-app/pnhechapfaindjhompbnflcldabbghjo'
                  footerText='Install Chrome Secure Shell'
                />
              )}
            </EuiFlexGroup>
          </EuiAccordion>
        </EuiPageContentBody>
      </EuiPageContent>
      <EuiSpacer />
      <EuiPageContent>
        <EuiPageContentBody>
          <EuiFlexGroup>
            <EuiFlexItem>
              <Iframe
                src={iframeUrl}
                height='1000px'
                width='100%'
                key={iframeUrl}
              />
            </EuiFlexItem>
          </EuiFlexGroup>
        </EuiPageContentBody>
      </EuiPageContent>
      <EuiGlobalToastList
        toasts={toasts}
        dismissToast={removeToast}
        toastLifeTimeMs={6000}
      />
    </EuiPageBody>
  )
}
