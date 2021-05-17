[![Community Project header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Project.png)](https://opensource.newrelic.com/oss-category/#community-project)

# New Relic Pixie Integration

The Pixie integration pulls a set of curated observability data from Pixie to send it to New Relic using the OpenTelemetry line protocol. The integration leverages PXL scripts to retrieve the data from Pixie.

## Getting Started

Make sure you have a Pixie account and a New Relic account set up, and have collected the following information:

 * [New Relic license key](https://docs.newrelic.com/docs/accounts/accounts-billing/account-setup/new-relic-license-key/)
 * [Pixie API key](https://docs.pixielabs.ai/using-pixie/api-quick-start/#get-an-api-token)
 * [Pixie Cluster ID](https://docs.pixielabs.ai/using-pixie/api-quick-start/#get-a-cluster-id)
 * Cluster name: the name of your Kubernetes cluster

## Building

```make build-container```

Docker is required to build the Pixie integration container image.

## Usage

```docker run --env-file ./env.list -it newrelic/newrelic-pixie-integration:latest```

Define the following environment variables in the `env.list` file:

```
CLUSTER_NAME=
PIXIE_API_KEY=
PIXIE_CLUSTER_ID=
NR_LICENSE_KEY=
```

The following environment variables are optional. 

```
PIXIE_ENDPOINT=work.withpixie.ai:443
NR_OTLP_HOST=otlp.nr-data.net:4317
VERBOSE=true
```

## Testing

After executing the command above Pixie data should be flowing into your New Relic account. Use the following NRQL queries to verify this:

**Metrics**
```
SELECT * FROM Metric WHERE instrumentation.provider='pixie'
```

**Spans**
```
SELECT * FROM Span WHERE instrumentation.provider='pixie'
```

## Support

New Relic hosts and moderates an online forum where customers can interact with New Relic employees as well as other users to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub. You can find this project's topic/threads here: https://discuss.newrelic.com/t/new-relic-pixie-integration/143646

## Contribute

We encourage your contributions to improve the Pixie integration! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA once per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you!  Without your contribution, this project would not be what it is today.

## License
The New Relic Pixie integration is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.

The integration also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the third-party notices document.
