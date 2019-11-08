import React from 'react'
import { EuiCard, EuiIcon, EuiFlexItem, EuiLink } from '@elastic/eui'
import PropTypes from 'prop-types'

export const Card = props => (
  <EuiFlexItem style={{ minWidth: 175 }}>
    <EuiCard
      layout='vertical'
      icon={<EuiIcon size='xxl' type={props.logo} />}
      title={props.title}
      description={props.desc}
      onClick={() => props.clickHandler(props.url)}
      isDisabled={props.disabled}
      footer={
        props.footerText &&
        props.footerUrl && (
          <EuiLink href={props.footerUrl} external target='_blank'>
            {props.footerText}
          </EuiLink>
        )
      }
    />
  </EuiFlexItem>
)

Card.propTypes = {
  logo: PropTypes.string,
  title: PropTypes.string,
  desc: PropTypes.string,
  url: PropTypes.string,
  disabled: PropTypes.bool,
  footerText: PropTypes.string,
  footerUrl: PropTypes.string,
  clickHandler: PropTypes.func
}
