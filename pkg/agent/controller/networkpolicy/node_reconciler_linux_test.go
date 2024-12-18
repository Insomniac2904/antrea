//go:build linux
// +build linux

// Copyright 2024 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package networkpolicy

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/sets"

	routetest "antrea.io/antrea/pkg/agent/route/testing"
	"antrea.io/antrea/pkg/apis/controlplane/v1beta2"
	secv1beta1 "antrea.io/antrea/pkg/apis/crd/v1beta1"
)

var (
	ruleActionAllow = secv1beta1.RuleActionAllow

	ipv4Net1 = newCIDR("192.168.1.0/24")
	ipv6Net1 = newCIDR("fec0::192:168:1:0/124")
	ipv4Net2 = newCIDR("192.168.1.128/25")
	ipv6Net2 = newCIDR("fec0::192:168:1:1/125")
	ipBlocks = v1beta2.NetworkPolicyPeer{
		IPBlocks: []v1beta2.IPBlock{
			{
				CIDR: v1beta2.IPNet{IP: v1beta2.IPAddress(ipv4Net1.IP), PrefixLength: 24},
				Except: []v1beta2.IPNet{
					{IP: v1beta2.IPAddress(ipv4Net2.IP), PrefixLength: 25},
				},
			},
			{
				CIDR: v1beta2.IPNet{IP: v1beta2.IPAddress(ipv6Net1.IP), PrefixLength: 124},
				Except: []v1beta2.IPNet{
					{IP: v1beta2.IPAddress(ipv6Net2.IP), PrefixLength: 125},
				},
			},
		},
	}
	ipBlocksToMatchAny = v1beta2.NetworkPolicyPeer{
		IPBlocks: []v1beta2.IPBlock{
			{
				CIDR: v1beta2.IPNet{IP: v1beta2.IPAddress(net.IPv4zero), PrefixLength: 0},
			},
			{
				CIDR: v1beta2.IPNet{IP: v1beta2.IPAddress(net.IPv4zero), PrefixLength: 0},
			},
		},
	}

	policyPriority1 = float64(1)
	tierPriority1   = int32(1)
	tierPriority2   = int32(2)

	ingressRuleID1 = "ingressRule1"
	ingressRuleID2 = "ingressRule2"
	ingressRuleID3 = "ingressRule3"
	egressRuleID1  = "egressRule1"
	egressRuleID2  = "egressRule2"
	ingressRule1   = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID1,
			Name:           "ingress-rule-01",
			PolicyName:     "ingress-policy",
			From:           ipBlocks,
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP80, serviceTCP443},
			Action:         &ruleActionAllow,
			Priority:       1,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority1,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "",
		},
		FromAddresses: dualAddressGroup1,
		ToAddresses:   nil,
	}
	ingressRule2 = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID2,
			Name:           "ingress-rule-02",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP443},
			Action:         &ruleActionAllow,
			Priority:       2,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  false,
			LogLabel:       "ingress2",
		},
		FromAddresses: dualAddressGroup1,
		ToAddresses:   nil,
	}
	ingressRule3 = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			From:           ipBlocksToMatchAny,
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: nil,
		ToAddresses:   nil,
	}
	ingressRule3WithFromAnyAddress        = ingressRule3
	updatedIngressRule3WithOneFromAddress = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: addressGroup1,
		ToAddresses:   nil,
	}
	updatedIngressRule3WithAnotherFromAddress = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: addressGroup2,
		ToAddresses:   nil,
	}
	updatedIngressRule3WithMultipleFromAddresses = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: addressGroup2.Union(addressGroup1),
		ToAddresses:   nil,
	}
	updatedIngressRule3WithOtherMultipleFromAddresses = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: addressGroup2.Union(v1beta2.NewGroupMemberSet(newAddressGroupMember("1.1.1.3"))),
		ToAddresses:   nil,
	}
	updatedIngressRule3WithFromNoAddress = &CompletedRule{
		rule: &rule{
			ID:             ingressRuleID3,
			Name:           "ingress-rule-03",
			PolicyName:     "ingress-policy",
			Direction:      v1beta2.DirectionIn,
			Services:       []v1beta2.Service{serviceTCP8080},
			Action:         &ruleActionAllow,
			Priority:       3,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "ingress3",
		},
		FromAddresses: nil,
		ToAddresses:   nil,
	}
	egressRule1 = &CompletedRule{
		rule: &rule{
			ID:             egressRuleID1,
			Name:           "egress-rule-01",
			PolicyName:     "egress-policy",
			Direction:      v1beta2.DirectionOut,
			Services:       []v1beta2.Service{serviceTCP80, serviceTCP443},
			Action:         &ruleActionAllow,
			Priority:       1,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority1,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "egress1",
		},
		ToAddresses:   dualAddressGroup1,
		FromAddresses: nil,
	}
	egressRule2 = &CompletedRule{
		rule: &rule{
			ID:             egressRuleID2,
			Name:           "egress-rule-02",
			PolicyName:     "egress-policy",
			Direction:      v1beta2.DirectionOut,
			Services:       []v1beta2.Service{serviceTCP443},
			Action:         &ruleActionAllow,
			Priority:       2,
			PolicyPriority: &policyPriority1,
			TierPriority:   &tierPriority2,
			SourceRef:      &cnp1,
			EnableLogging:  true,
			LogLabel:       "egress2:test_log_label",
		},
		ToAddresses:   dualAddressGroup1,
		FromAddresses: nil,
	}
)

func newTestNodeReconciler(mockRouteClient *routetest.MockInterface, ipv4Enabled, ipv6Enabled bool) *nodeReconciler {
	return newNodeReconciler(mockRouteClient, ipv4Enabled, ipv6Enabled)
}

func TestNodeReconcilerReconcileAndForget(t *testing.T) {
	tests := []struct {
		name          string
		rulesToAdd    []*CompletedRule
		rulesToForget []string
		ipv4Enabled   bool
		ipv6Enabled   bool
		expectedCalls func(mockRouteClient *routetest.MockInterfaceMockRecorder)
	}{
		{
			name:        "IPv4, add an ingress rule, update it, then forget it",
			ipv4Enabled: true,
			ipv6Enabled: false,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				serviceRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, [][]string{nil}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
			},
			rulesToAdd: []*CompletedRule{
				ingressRule1,
			},
			rulesToForget: []string{
				ingressRuleID1,
			},
		},
		{
			name:        "IPv6, add an egress rule, then forget it",
			ipv4Enabled: false,
			ipv6Enabled: true,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				serviceRules := [][]string{
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESSRULE1"}, serviceRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESS-RULES"}, coreRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESS-RULES"}, [][]string{nil}, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESSRULE1"}, true),
				)
			},
			rulesToAdd: []*CompletedRule{
				egressRule1,
			},
			rulesToForget: []string{
				egressRuleID1,
			},
		},
		{
			name:        "Dualstack, add an ingress rule, then forget it",
			ipv4Enabled: true,
			ipv6Enabled: true,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				serviceRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRulesIPv4 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesIPv6 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-6 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesIPv4, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, [][]string{nil}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", sets.New[string]("2002:1a23:fb44::1/128", "fec0::192:168:1:8/125"), true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesIPv6, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, [][]string{nil}, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", true),
				)
			},
			rulesToAdd: []*CompletedRule{
				ingressRule1,
			},
			rulesToForget: []string{
				ingressRuleID1,
			},
		},
		{
			name:        "IPv4, add multiple ingress rules whose priorities are in ascending order, then forget some",
			ipv4Enabled: true,
			ipv6Enabled: false,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				serviceRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDeleted3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDelete2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules2, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDeleted3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDelete2, false),
				)

			},
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
				ingressRule3,
			},
			rulesToForget: []string{
				ingressRuleID3,
				ingressRuleID2,
			},
		},
		{
			name:        "IPv4, add multiple ingress rules whose priorities are in descending order, then forget some",
			ipv4Enabled: true,
			ipv6Enabled: false,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreRules3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				serviceRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDelete3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDelete1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules2, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDelete3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDelete1, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
			},
			rulesToAdd: []*CompletedRule{
				ingressRule3,
				ingressRule2,
				ingressRule1,
			},
			rulesToForget: []string{
				ingressRuleID3,
				ingressRuleID1,
			},
		},
		{
			name:        "IPv4, add multiple ingress rules whose priorities are in random order, then forget some",
			ipv4Enabled: true,
			ipv6Enabled: false,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreRules2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				serviceRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				coreRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDelete2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRulesDelete1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules2, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, serviceRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDelete2, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRulesDelete1, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
			},
			rulesToAdd: []*CompletedRule{
				ingressRule2,
				ingressRule1,
				ingressRule3,
			},
			rulesToForget: []string{
				ingressRuleID2,
				ingressRuleID1,
			},
		},
		{
			name:        "IPv4, add an ingress rule, then update it several times, forget it finally",
			ipv4Enabled: true,
			ipv6Enabled: false,
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreRules1 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules2 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules3 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.2/32 -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.2/32 -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules4 := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE3-4 src -p tcp --dport 8080 -j LOG --log-prefix "Antrea:I:Allow:ingress3:"`,
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE3-4 src -p tcp --dport 8080 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-03, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				coreRules6 := coreRules2
				coreRules7 := coreRules1
				coreRules8 := coreRules4
				coreRules9 := coreRules1
				coreRules10 := [][]string{nil}
				coreRules11 := coreRules4
				coreRules12 := coreRules10
				coreRules13 := coreRules1
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules1, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules2, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules3, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", sets.New[string]("1.1.1.1/32", "1.1.1.2/32"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules4, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", sets.New[string]("1.1.1.2/32", "1.1.1.3/32"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules6, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules7, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", sets.New[string]("1.1.1.1/32", "1.1.1.2/32"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules8, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules9, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules10, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", sets.New[string]("1.1.1.1/32", "1.1.1.2/32"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules11, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules12, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE3-4", false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, coreRules13, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESS-RULES"}, [][]string{nil}, false),
				)
			},
			rulesToAdd: []*CompletedRule{
				ingressRule3WithFromAnyAddress,
				updatedIngressRule3WithOneFromAddress,
				updatedIngressRule3WithAnotherFromAddress,
				updatedIngressRule3WithMultipleFromAddresses,
				updatedIngressRule3WithOtherMultipleFromAddresses,
				updatedIngressRule3WithOneFromAddress,
				ingressRule3WithFromAnyAddress,
				updatedIngressRule3WithMultipleFromAddresses,
				ingressRule3WithFromAnyAddress,
				updatedIngressRule3WithFromNoAddress,
				updatedIngressRule3WithMultipleFromAddresses,
				updatedIngressRule3WithFromNoAddress,
				ingressRule3WithFromAnyAddress,
			},
			rulesToForget: []string{
				ingressRuleID3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			mockRouteClient := routetest.NewMockInterface(controller)
			r := newTestNodeReconciler(mockRouteClient, tt.ipv4Enabled, tt.ipv6Enabled)

			tt.expectedCalls(mockRouteClient.EXPECT())
			for _, rule := range tt.rulesToAdd {
				assert.NoError(t, r.Reconcile(rule))
			}
			for _, rule := range tt.rulesToForget {
				assert.NoError(t, r.Forget(rule))
			}
		})
	}
}

func TestNodeReconcilerBatchReconcileAndForget(t *testing.T) {
	tests := []struct {
		name          string
		ipv4Enabled   bool
		ipv6Enabled   bool
		rulesToAdd    []*CompletedRule
		rulesToForget []string
		expectedCalls func(mockRouteClient *routetest.MockInterfaceMockRecorder)
	}{
		{
			name:        "IPv4, add ingress rules in batch, then forget one",
			ipv4Enabled: true,
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
			},
			rulesToForget: []string{
				ingressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-INGRESS-RULES",
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-INGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				updatedCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, coreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedCoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables(svcChains, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
			},
		},
		{
			name:        "IPv6, add ingress rules in batch, then forget one",
			ipv6Enabled: true,
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
			},
			rulesToForget: []string{
				ingressRuleID2,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-INGRESS-RULES",
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-6 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-INGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				updatedCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-6 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", sets.New[string]("2002:1a23:fb44::1/128", "fec0::192:168:1:8/125"), true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, coreRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedCoreRules, true),
				)
			},
		},
		{
			name:        "dualstack, add ingress rules in batch, then forget one",
			ipv4Enabled: true,
			ipv6Enabled: true,
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
			},
			rulesToForget: []string{
				ingressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-INGRESS-RULES",
				}
				ipv4CoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				ipv6CoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-6 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-INGRESSRULE1",
				}
				ipv4SvcRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				ipv6SvcRules := ipv4SvcRules
				updatedIPv4CoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedIPv6CoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}

				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, ipv4SvcRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, ipv4CoreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedIPv4CoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
				)
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", sets.New[string]("2002:1a23:fb44::1/128", "fec0::192:168:1:8/125"), true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, ipv6SvcRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, ipv6CoreRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedIPv6CoreRules, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", true),
				)
			},
		},
		{
			name:        "IPv4, add egress rules in batch, then forget one",
			ipv4Enabled: true,
			rulesToAdd: []*CompletedRule{
				egressRule1,
				egressRule2,
			},
			rulesToForget: []string{
				egressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-EGRESS-RULES",
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-EGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				updatedCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, coreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedCoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables(svcChains, false),
				)
			},
		},
		{
			name:        "IPv6, add egress rules in batch, then forget one",
			ipv6Enabled: true,
			rulesToAdd: []*CompletedRule{
				egressRule1,
				egressRule2,
			},
			rulesToForget: []string{
				egressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-EGRESS-RULES",
				}
				coreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-EGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				updatedCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, coreRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedCoreRules, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables(svcChains, true),
				)
			},
		},
		{
			name:        "dualstack, only add egress rules, then forget one",
			ipv4Enabled: true,
			ipv6Enabled: true,
			rulesToAdd: []*CompletedRule{
				egressRule1,
				egressRule2,
			},
			rulesToForget: []string{
				egressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				coreChains := []string{
					"ANTREA-POL-EGRESS-RULES",
				}
				ipv4CoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				ipv6CoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				svcChains := []string{
					"ANTREA-POL-EGRESSRULE1",
				}
				ipv4SvcRules := [][]string{
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				ipv6SvcRules := ipv4SvcRules
				updatedIPv4CoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedIPv6CoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, ipv4SvcRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, ipv4CoreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, ipv6SvcRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, ipv6CoreRules, true),

					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedIPv4CoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables(svcChains, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(coreChains, updatedIPv6CoreRules, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables(svcChains, true),
				)
			},
		},
		{
			name:        "IPv4, add ingress and egress rules in batch, then forget some rules",
			ipv4Enabled: true,
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
				egressRule1,
				egressRule2,
			},
			rulesToForget: []string{
				ingressRuleID1,
				egressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				svcChains := []string{
					"ANTREA-POL-INGRESSRULE1",
					"ANTREA-POL-EGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				ingressCoreChains := []string{"ANTREA-POL-INGRESS-RULES"}
				egressCoreChains := []string{"ANTREA-POL-EGRESS-RULES"}
				ingressCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-4 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				egressCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedIngressCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedEgressCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 1.1.1.1/32 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", sets.New[string]("1.1.1.1/32", "192.168.1.0/25"), false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(ingressCoreChains, ingressCoreRules, false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(egressCoreChains, egressCoreRules, false),

					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(ingressCoreChains, updatedIngressCoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-4", false),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(egressCoreChains, updatedEgressCoreRules, false),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESSRULE1"}, false),
				)
			},
		},
		{
			name:        "IPv6, add ingress and egress rules in batch, then forget some rules",
			ipv6Enabled: true,
			rulesToAdd: []*CompletedRule{
				ingressRule1,
				ingressRule2,
				egressRule1,
				egressRule2,
			},
			rulesToForget: []string{
				ingressRuleID1,
				egressRuleID1,
			},
			expectedCalls: func(mockRouteClient *routetest.MockInterfaceMockRecorder) {
				svcChains := []string{
					"ANTREA-POL-INGRESSRULE1",
					"ANTREA-POL-EGRESSRULE1",
				}
				svcRules := [][]string{
					{
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:I:Allow:"`,
						`-A ANTREA-POL-INGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
					{
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 80 -j ACCEPT`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress1:"`,
						`-A ANTREA-POL-EGRESSRULE1 -p tcp --dport 443 -j ACCEPT`,
					},
				}
				ingressCoreChains := []string{"ANTREA-POL-INGRESS-RULES"}
				egressCoreChains := []string{"ANTREA-POL-EGRESS-RULES"}
				ingressCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -m set --match-set ANTREA-POL-INGRESSRULE1-6 src -j ANTREA-POL-INGRESSRULE1 -m comment --comment "Antrea: for rule ingress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-INGRESS-RULES -s 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				egressCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -j ANTREA-POL-EGRESSRULE1 -m comment --comment "Antrea: for rule egress-rule-01, policy AntreaClusterNetworkPolicy:name1"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedIngressCoreRules := [][]string{
					{
						`-A ANTREA-POL-INGRESS-RULES -s 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule ingress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				updatedEgressCoreRules := [][]string{
					{
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j LOG --log-prefix "Antrea:O:Allow:egress2:test:"`,
						`-A ANTREA-POL-EGRESS-RULES -d 2002:1a23:fb44::1/128 -p tcp --dport 443 -j ACCEPT -m comment --comment "Antrea: for rule egress-rule-02, policy AntreaClusterNetworkPolicy:name1"`,
					},
				}
				gomock.InOrder(
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", sets.New[string]("2002:1a23:fb44::1/128", "fec0::192:168:1:8/125"), true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(svcChains, svcRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(ingressCoreChains, ingressCoreRules, true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(egressCoreChains, egressCoreRules, true),

					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(ingressCoreChains, updatedIngressCoreRules, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-INGRESSRULE1"}, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPSet("ANTREA-POL-INGRESSRULE1-6", true),
					mockRouteClient.AddOrUpdateNodeNetworkPolicyIPTables(egressCoreChains, updatedEgressCoreRules, true),
					mockRouteClient.DeleteNodeNetworkPolicyIPTables([]string{"ANTREA-POL-EGRESSRULE1"}, true),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			mockRouteClient := routetest.NewMockInterface(controller)
			r := newTestNodeReconciler(mockRouteClient, tt.ipv4Enabled, tt.ipv6Enabled)

			tt.expectedCalls(mockRouteClient.EXPECT())
			assert.NoError(t, r.BatchReconcile(tt.rulesToAdd))

			for _, ruleID := range tt.rulesToForget {
				assert.NoError(t, r.Forget(ruleID))
			}
		})
	}
}
