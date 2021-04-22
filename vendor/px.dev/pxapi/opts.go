/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package pxapi

// ClientOption configures options on the client.
type ClientOption func(client *Client)

// WithCloudAddr is the option to specify cloud address to use.
func WithCloudAddr(cloudAddr string) ClientOption {
	return func(c *Client) {
		c.cloudAddr = cloudAddr
	}
}

// WithBearerAuth is the option to specify bearer auth to use.
func WithBearerAuth(auth string) ClientOption {
	return func(c *Client) {
		c.bearerAuth = auth
	}
}

// WithAPIKey is the option to specify the API key to use.
func WithAPIKey(auth string) ClientOption {
	return func(c *Client) {
		c.apiKey = auth
	}
}
