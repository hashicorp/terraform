package akamai

import "github.com/hashicorp/terraform/helper/schema"

var akamaiPropertySchema map[string]*schema.Schema = map[string]*schema.Schema{
	"clone_from": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"property_id": {
					Type:     schema.TypeString,
					Required: true,
				},
				"version": {
					Type:     schema.TypeInt,
					Optional: true,
				},
				"etag": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"copy_hostnames": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	},
	"group": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"contract": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"product_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"cpcode": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"name": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"ipv6": &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	},
	"hostname": &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"contact": &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"edge_hostname": &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"origin": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"is_secure": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"hostname": {
					Type:     schema.TypeString,
					Required: true,
				},

				"port": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  80,
				},

				"forward_hostname": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "ORIGIN_HOSTNAME",
				},
			},
		},
	},

	"compress": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"extensions": {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				"content_types": {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				"criteria": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"option": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"values": {
											Type:     schema.TypeSet,
											Elem:     &schema.Schema{Type: schema.TypeString},
											Optional: true,
										},
										"value": {
											Type:     schema.TypeString,
											Optional: true,
										},
										"flag": {
											Type:     schema.TypeBool,
											Optional: true,
										},
										"type": {
											Type:     schema.TypeString,
											Optional: true,
											Default:  "auto",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},

	"cache": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"match": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"extensions": {
								Type:     schema.TypeSet,
								Elem:     &schema.Schema{Type: schema.TypeString},
								Optional: true,
							},
							"paths": {
								Type:     schema.TypeSet,
								Elem:     &schema.Schema{Type: schema.TypeString},
								Optional: true,
							},
						},
					},
				},
				"max_age": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"prefreshing": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"prefetch": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"query_params": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"query_params_sort": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"cache": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"criteria": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"option": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"values": {
											Type:     schema.TypeSet,
											Elem:     &schema.Schema{Type: schema.TypeString},
											Optional: true,
										},
										"value": {
											Type:     schema.TypeString,
											Optional: true,
										},
										"flag": {
											Type:     schema.TypeBool,
											Optional: true,
										},
										"type": {
											Type:     schema.TypeString,
											Optional: true,
											Default:  "auto",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},

	"rule": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"comment": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"criteria_match": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "all",
				},
				"criteria": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"option": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"values": {
											Type:     schema.TypeSet,
											Elem:     &schema.Schema{Type: schema.TypeString},
											Optional: true,
										},
										"value": {
											Type:     schema.TypeString,
											Optional: true,
										},
										"flag": {
											Type:     schema.TypeBool,
											Optional: true,
										},
										"type": {
											Type:     schema.TypeString,
											Optional: true,
											Default:  "auto",
										},
									},
								},
							},
						},
					},
				},
				"behavior": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"option": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"values": {
											Type:     schema.TypeSet,
											Elem:     &schema.Schema{Type: schema.TypeString},
											Optional: true,
										},
										"value": {
											Type:     schema.TypeString,
											Optional: true,
										},
										"flag": {
											Type:     schema.TypeBool,
											Optional: true,
										},
										"type": {
											Type:     schema.TypeString,
											Optional: true,
											Default:  "auto",
										},
									},
								},
							},
						},
					},
				},
				"rule": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"comment": {
								Type:     schema.TypeString,
								Required: true,
							},
							"criteria_match": {
								Type:     schema.TypeString,
								Optional: true,
								Default:  "all",
							},
							"criteria": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"option": {
											Type:     schema.TypeSet,
											Optional: true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"name": {
														Type:     schema.TypeString,
														Required: true,
													},
													"values": {
														Type:     schema.TypeSet,
														Elem:     &schema.Schema{Type: schema.TypeString},
														Optional: true,
													},
													"value": {
														Type:     schema.TypeString,
														Optional: true,
													},
													"flag": {
														Type:     schema.TypeBool,
														Optional: true,
													},
													"type": {
														Type:     schema.TypeString,
														Optional: true,
														Default:  "auto",
													},
												},
											},
										},
									},
								},
							},
							"behavior": {
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"option": {
											Type:     schema.TypeSet,
											Optional: true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"name": {
														Type:     schema.TypeString,
														Required: true,
													},
													"values": {
														Type:     schema.TypeSet,
														Elem:     &schema.Schema{Type: schema.TypeString},
														Optional: true,
													},
													"value": {
														Type:     schema.TypeString,
														Optional: true,
													},
													"flag": {
														Type:     schema.TypeBool,
														Optional: true,
													},
													"type": {
														Type:     schema.TypeString,
														Optional: true,
														Default:  "auto",
													},
												},
											},
										},
									},
								},
							},
							"rule": &schema.Schema{
								Type:     schema.TypeSet,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"comment": {
											Type:     schema.TypeString,
											Required: true,
										},
										"criteria_match": {
											Type:     schema.TypeString,
											Optional: true,
											Default:  "all",
										},
										"criteria": {
											Type:     schema.TypeSet,
											Optional: true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"name": {
														Type:     schema.TypeString,
														Required: true,
													},
													"option": {
														Type:     schema.TypeSet,
														Optional: true,
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"values": {
																	Type:     schema.TypeSet,
																	Elem:     &schema.Schema{Type: schema.TypeString},
																	Optional: true,
																},
																"value": {
																	Type:     schema.TypeString,
																	Optional: true,
																},
																"flag": {
																	Type:     schema.TypeBool,
																	Optional: true,
																},
																"type": {
																	Type:     schema.TypeString,
																	Optional: true,
																	Default:  "auto",
																},
															},
														},
													},
												},
											},
										},
										"behavior": {
											Type:     schema.TypeSet,
											Optional: true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"name": {
														Type:     schema.TypeString,
														Required: true,
													},
													"option": {
														Type:     schema.TypeSet,
														Optional: true,
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"values": {
																	Type:     schema.TypeSet,
																	Elem:     &schema.Schema{Type: schema.TypeString},
																	Optional: true,
																},
																"value": {
																	Type:     schema.TypeString,
																	Optional: true,
																},
																"flag": {
																	Type:     schema.TypeBool,
																	Optional: true,
																},
																"type": {
																	Type:     schema.TypeString,
																	Optional: true,
																	Default:  "auto",
																},
															},
														},
													},
												},
											},
										},
										"rule": &schema.Schema{
											Type:     schema.TypeSet,
											Optional: true,
											Elem: &schema.Resource{
												Schema: map[string]*schema.Schema{
													"name": {
														Type:     schema.TypeString,
														Required: true,
													},
													"comment": {
														Type:     schema.TypeString,
														Required: true,
													},
													"criteria_match": {
														Type:     schema.TypeString,
														Optional: true,
														Default:  "all",
													},
													"criteria": {
														Type:     schema.TypeSet,
														Optional: true,
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"option": {
																	Type:     schema.TypeSet,
																	Optional: true,
																	Elem: &schema.Resource{
																		Schema: map[string]*schema.Schema{
																			"name": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"values": {
																				Type:     schema.TypeSet,
																				Elem:     &schema.Schema{Type: schema.TypeString},
																				Optional: true,
																			},
																			"value": {
																				Type:     schema.TypeString,
																				Optional: true,
																			},
																			"flag": {
																				Type:     schema.TypeBool,
																				Optional: true,
																			},
																			"type": {
																				Type:     schema.TypeString,
																				Optional: true,
																				Default:  "auto",
																			},
																		},
																	},
																},
															},
														},
													},
													"behavior": {
														Type:     schema.TypeSet,
														Optional: true,
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"option": {
																	Type:     schema.TypeSet,
																	Optional: true,
																	Elem: &schema.Resource{
																		Schema: map[string]*schema.Schema{
																			"name": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"values": {
																				Type:     schema.TypeSet,
																				Elem:     &schema.Schema{Type: schema.TypeString},
																				Optional: true,
																			},
																			"value": {
																				Type:     schema.TypeString,
																				Optional: true,
																			},
																			"flag": {
																				Type:     schema.TypeBool,
																				Optional: true,
																			},
																			"type": {
																				Type:     schema.TypeString,
																				Optional: true,
																				Default:  "auto",
																			},
																		},
																	},
																},
															},
														},
													},
													"rule": &schema.Schema{
														Type:     schema.TypeSet,
														Optional: true,
														Elem: &schema.Resource{
															Schema: map[string]*schema.Schema{
																"name": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"comment": {
																	Type:     schema.TypeString,
																	Required: true,
																},
																"criteria_match": {
																	Type:     schema.TypeString,
																	Optional: true,
																	Default:  "all",
																},
																"criteria": {
																	Type:     schema.TypeSet,
																	Optional: true,
																	Elem: &schema.Resource{
																		Schema: map[string]*schema.Schema{
																			"name": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"option": {
																				Type:     schema.TypeSet,
																				Optional: true,
																				Elem: &schema.Resource{
																					Schema: map[string]*schema.Schema{
																						"name": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"values": {
																							Type:     schema.TypeSet,
																							Elem:     &schema.Schema{Type: schema.TypeString},
																							Optional: true,
																						},
																						"value": {
																							Type:     schema.TypeString,
																							Optional: true,
																						},
																						"flag": {
																							Type:     schema.TypeBool,
																							Optional: true,
																						},
																						"type": {
																							Type:     schema.TypeString,
																							Optional: true,
																							Default:  "auto",
																						},
																					},
																				},
																			},
																		},
																	},
																},
																"behavior": {
																	Type:     schema.TypeSet,
																	Optional: true,
																	Elem: &schema.Resource{
																		Schema: map[string]*schema.Schema{
																			"name": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"option": {
																				Type:     schema.TypeSet,
																				Optional: true,
																				Elem: &schema.Resource{
																					Schema: map[string]*schema.Schema{
																						"name": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"values": {
																							Type:     schema.TypeSet,
																							Elem:     &schema.Schema{Type: schema.TypeString},
																							Optional: true,
																						},
																						"value": {
																							Type:     schema.TypeString,
																							Optional: true,
																						},
																						"flag": {
																							Type:     schema.TypeBool,
																							Optional: true,
																						},
																						"type": {
																							Type:     schema.TypeString,
																							Optional: true,
																							Default:  "auto",
																						},
																					},
																				},
																			},
																		},
																	},
																},
																"rule": &schema.Schema{
																	Type:     schema.TypeSet,
																	Optional: true,
																	Elem: &schema.Resource{
																		Schema: map[string]*schema.Schema{
																			"name": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"comment": {
																				Type:     schema.TypeString,
																				Required: true,
																			},
																			"criteria_match": {
																				Type:     schema.TypeString,
																				Optional: true,
																				Default:  "all",
																			},
																			"criteria": {
																				Type:     schema.TypeSet,
																				Optional: true,
																				Elem: &schema.Resource{
																					Schema: map[string]*schema.Schema{
																						"name": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"option": {
																							Type:     schema.TypeSet,
																							Optional: true,
																							Elem: &schema.Resource{
																								Schema: map[string]*schema.Schema{
																									"name": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"values": {
																										Type:     schema.TypeSet,
																										Elem:     &schema.Schema{Type: schema.TypeString},
																										Optional: true,
																									},
																									"value": {
																										Type:     schema.TypeString,
																										Optional: true,
																									},
																									"flag": {
																										Type:     schema.TypeBool,
																										Optional: true,
																									},
																									"type": {
																										Type:     schema.TypeString,
																										Optional: true,
																										Default:  "auto",
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			"behavior": {
																				Type:     schema.TypeSet,
																				Optional: true,
																				Elem: &schema.Resource{
																					Schema: map[string]*schema.Schema{
																						"name": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"option": {
																							Type:     schema.TypeSet,
																							Optional: true,
																							Elem: &schema.Resource{
																								Schema: map[string]*schema.Schema{
																									"name": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"values": {
																										Type:     schema.TypeSet,
																										Elem:     &schema.Schema{Type: schema.TypeString},
																										Optional: true,
																									},
																									"value": {
																										Type:     schema.TypeString,
																										Optional: true,
																									},
																									"flag": {
																										Type:     schema.TypeBool,
																										Optional: true,
																									},
																									"type": {
																										Type:     schema.TypeString,
																										Optional: true,
																										Default:  "auto",
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			"rule": &schema.Schema{
																				Type:     schema.TypeSet,
																				Optional: true,
																				Elem: &schema.Resource{
																					Schema: map[string]*schema.Schema{
																						"name": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"comment": {
																							Type:     schema.TypeString,
																							Required: true,
																						},
																						"criteria_match": {
																							Type:     schema.TypeString,
																							Optional: true,
																							Default:  "all",
																						},
																						"criteria": {
																							Type:     schema.TypeSet,
																							Optional: true,
																							Elem: &schema.Resource{
																								Schema: map[string]*schema.Schema{
																									"name": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"option": {
																										Type:     schema.TypeSet,
																										Optional: true,
																										Elem: &schema.Resource{
																											Schema: map[string]*schema.Schema{
																												"name": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"values": {
																													Type:     schema.TypeSet,
																													Elem:     &schema.Schema{Type: schema.TypeString},
																													Optional: true,
																												},
																												"value": {
																													Type:     schema.TypeString,
																													Optional: true,
																												},
																												"flag": {
																													Type:     schema.TypeBool,
																													Optional: true,
																												},
																												"type": {
																													Type:     schema.TypeString,
																													Optional: true,
																													Default:  "auto",
																												},
																											},
																										},
																									},
																								},
																							},
																						},
																						"behavior": {
																							Type:     schema.TypeSet,
																							Optional: true,
																							Elem: &schema.Resource{
																								Schema: map[string]*schema.Schema{
																									"name": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"option": {
																										Type:     schema.TypeSet,
																										Optional: true,
																										Elem: &schema.Resource{
																											Schema: map[string]*schema.Schema{
																												"name": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"values": {
																													Type:     schema.TypeSet,
																													Elem:     &schema.Schema{Type: schema.TypeString},
																													Optional: true,
																												},
																												"value": {
																													Type:     schema.TypeString,
																													Optional: true,
																												},
																												"flag": {
																													Type:     schema.TypeBool,
																													Optional: true,
																												},
																												"type": {
																													Type:     schema.TypeString,
																													Optional: true,
																													Default:  "auto",
																												},
																											},
																										},
																									},
																								},
																							},
																						},
																						"rule": &schema.Schema{
																							Type:     schema.TypeSet,
																							Optional: true,
																							Elem: &schema.Resource{
																								Schema: map[string]*schema.Schema{
																									"name": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"comment": {
																										Type:     schema.TypeString,
																										Required: true,
																									},
																									"criteria_match": {
																										Type:     schema.TypeString,
																										Optional: true,
																										Default:  "all",
																									},
																									"criteria": {
																										Type:     schema.TypeSet,
																										Optional: true,
																										Elem: &schema.Resource{
																											Schema: map[string]*schema.Schema{
																												"name": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"option": {
																													Type:     schema.TypeSet,
																													Optional: true,
																													Elem: &schema.Resource{
																														Schema: map[string]*schema.Schema{
																															"name": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"values": {
																																Type:     schema.TypeSet,
																																Elem:     &schema.Schema{Type: schema.TypeString},
																																Optional: true,
																															},
																															"value": {
																																Type:     schema.TypeString,
																																Optional: true,
																															},
																															"flag": {
																																Type:     schema.TypeBool,
																																Optional: true,
																															},
																															"type": {
																																Type:     schema.TypeString,
																																Optional: true,
																																Default:  "auto",
																															},
																														},
																													},
																												},
																											},
																										},
																									},
																									"behavior": {
																										Type:     schema.TypeSet,
																										Optional: true,
																										Elem: &schema.Resource{
																											Schema: map[string]*schema.Schema{
																												"name": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"option": {
																													Type:     schema.TypeSet,
																													Optional: true,
																													Elem: &schema.Resource{
																														Schema: map[string]*schema.Schema{
																															"name": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"values": {
																																Type:     schema.TypeSet,
																																Elem:     &schema.Schema{Type: schema.TypeString},
																																Optional: true,
																															},
																															"value": {
																																Type:     schema.TypeString,
																																Optional: true,
																															},
																															"flag": {
																																Type:     schema.TypeBool,
																																Optional: true,
																															},
																															"type": {
																																Type:     schema.TypeString,
																																Optional: true,
																																Default:  "auto",
																															},
																														},
																													},
																												},
																											},
																										},
																									},
																									"rule": &schema.Schema{
																										Type:     schema.TypeSet,
																										Optional: true,
																										Elem: &schema.Resource{
																											Schema: map[string]*schema.Schema{
																												"name": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"comment": {
																													Type:     schema.TypeString,
																													Required: true,
																												},
																												"criteria_match": {
																													Type:     schema.TypeString,
																													Optional: true,
																													Default:  "all",
																												},
																												"criteria": {
																													Type:     schema.TypeSet,
																													Optional: true,
																													Elem: &schema.Resource{
																														Schema: map[string]*schema.Schema{
																															"name": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"option": {
																																Type:     schema.TypeSet,
																																Optional: true,
																																Elem: &schema.Resource{
																																	Schema: map[string]*schema.Schema{
																																		"name": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"values": {
																																			Type:     schema.TypeSet,
																																			Elem:     &schema.Schema{Type: schema.TypeString},
																																			Optional: true,
																																		},
																																		"value": {
																																			Type:     schema.TypeString,
																																			Optional: true,
																																		},
																																		"flag": {
																																			Type:     schema.TypeBool,
																																			Optional: true,
																																		},
																																		"type": {
																																			Type:     schema.TypeString,
																																			Optional: true,
																																			Default:  "auto",
																																		},
																																	},
																																},
																															},
																														},
																													},
																												},
																												"behavior": {
																													Type:     schema.TypeSet,
																													Optional: true,
																													Elem: &schema.Resource{
																														Schema: map[string]*schema.Schema{
																															"name": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"option": {
																																Type:     schema.TypeSet,
																																Optional: true,
																																Elem: &schema.Resource{
																																	Schema: map[string]*schema.Schema{
																																		"name": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"values": {
																																			Type:     schema.TypeSet,
																																			Elem:     &schema.Schema{Type: schema.TypeString},
																																			Optional: true,
																																		},
																																		"value": {
																																			Type:     schema.TypeString,
																																			Optional: true,
																																		},
																																		"flag": {
																																			Type:     schema.TypeBool,
																																			Optional: true,
																																		},
																																		"type": {
																																			Type:     schema.TypeString,
																																			Optional: true,
																																			Default:  "auto",
																																		},
																																	},
																																},
																															},
																														},
																													},
																												},
																												"rule": &schema.Schema{
																													Type:     schema.TypeSet,
																													Optional: true,
																													Elem: &schema.Resource{
																														Schema: map[string]*schema.Schema{
																															"name": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"comment": {
																																Type:     schema.TypeString,
																																Required: true,
																															},
																															"criteria_match": {
																																Type:     schema.TypeString,
																																Optional: true,
																																Default:  "all",
																															},
																															"criteria": {
																																Type:     schema.TypeSet,
																																Optional: true,
																																Elem: &schema.Resource{
																																	Schema: map[string]*schema.Schema{
																																		"name": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"option": {
																																			Type:     schema.TypeSet,
																																			Optional: true,
																																			Elem: &schema.Resource{
																																				Schema: map[string]*schema.Schema{
																																					"name": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"values": {
																																						Type:     schema.TypeSet,
																																						Elem:     &schema.Schema{Type: schema.TypeString},
																																						Optional: true,
																																					},
																																					"value": {
																																						Type:     schema.TypeString,
																																						Optional: true,
																																					},
																																					"flag": {
																																						Type:     schema.TypeBool,
																																						Optional: true,
																																					},
																																					"type": {
																																						Type:     schema.TypeString,
																																						Optional: true,
																																						Default:  "auto",
																																					},
																																				},
																																			},
																																		},
																																	},
																																},
																															},
																															"behavior": {
																																Type:     schema.TypeSet,
																																Optional: true,
																																Elem: &schema.Resource{
																																	Schema: map[string]*schema.Schema{
																																		"name": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"option": {
																																			Type:     schema.TypeSet,
																																			Optional: true,
																																			Elem: &schema.Resource{
																																				Schema: map[string]*schema.Schema{
																																					"name": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"values": {
																																						Type:     schema.TypeSet,
																																						Elem:     &schema.Schema{Type: schema.TypeString},
																																						Optional: true,
																																					},
																																					"value": {
																																						Type:     schema.TypeString,
																																						Optional: true,
																																					},
																																					"flag": {
																																						Type:     schema.TypeBool,
																																						Optional: true,
																																					},
																																					"type": {
																																						Type:     schema.TypeString,
																																						Optional: true,
																																						Default:  "auto",
																																					},
																																				},
																																			},
																																		},
																																	},
																																},
																															},
																															"rule": &schema.Schema{
																																Type:     schema.TypeSet,
																																Optional: true,
																																Elem: &schema.Resource{
																																	Schema: map[string]*schema.Schema{
																																		"name": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"comment": {
																																			Type:     schema.TypeString,
																																			Required: true,
																																		},
																																		"criteria_match": {
																																			Type:     schema.TypeString,
																																			Optional: true,
																																			Default:  "all",
																																		},
																																		"criteria": {
																																			Type:     schema.TypeSet,
																																			Optional: true,
																																			Elem: &schema.Resource{
																																				Schema: map[string]*schema.Schema{
																																					"name": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"option": {
																																						Type:     schema.TypeSet,
																																						Optional: true,
																																						Elem: &schema.Resource{
																																							Schema: map[string]*schema.Schema{
																																								"name": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"values": {
																																									Type:     schema.TypeSet,
																																									Elem:     &schema.Schema{Type: schema.TypeString},
																																									Optional: true,
																																								},
																																								"value": {
																																									Type:     schema.TypeString,
																																									Optional: true,
																																								},
																																								"flag": {
																																									Type:     schema.TypeBool,
																																									Optional: true,
																																								},
																																								"type": {
																																									Type:     schema.TypeString,
																																									Optional: true,
																																									Default:  "auto",
																																								},
																																							},
																																						},
																																					},
																																				},
																																			},
																																		},
																																		"behavior": {
																																			Type:     schema.TypeSet,
																																			Optional: true,
																																			Elem: &schema.Resource{
																																				Schema: map[string]*schema.Schema{
																																					"name": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"option": {
																																						Type:     schema.TypeSet,
																																						Optional: true,
																																						Elem: &schema.Resource{
																																							Schema: map[string]*schema.Schema{
																																								"name": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"values": {
																																									Type:     schema.TypeSet,
																																									Elem:     &schema.Schema{Type: schema.TypeString},
																																									Optional: true,
																																								},
																																								"value": {
																																									Type:     schema.TypeString,
																																									Optional: true,
																																								},
																																								"flag": {
																																									Type:     schema.TypeBool,
																																									Optional: true,
																																								},
																																								"type": {
																																									Type:     schema.TypeString,
																																									Optional: true,
																																									Default:  "auto",
																																								},
																																							},
																																						},
																																					},
																																				},
																																			},
																																		},
																																		"rule": &schema.Schema{
																																			Type:     schema.TypeSet,
																																			Optional: true,
																																			Elem: &schema.Resource{
																																				Schema: map[string]*schema.Schema{
																																					"name": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"comment": {
																																						Type:     schema.TypeString,
																																						Required: true,
																																					},
																																					"criteria_match": {
																																						Type:     schema.TypeString,
																																						Optional: true,
																																						Default:  "all",
																																					},
																																					"criteria": {
																																						Type:     schema.TypeSet,
																																						Optional: true,
																																						Elem: &schema.Resource{
																																							Schema: map[string]*schema.Schema{
																																								"name": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"option": {
																																									Type:     schema.TypeSet,
																																									Optional: true,
																																									Elem: &schema.Resource{
																																										Schema: map[string]*schema.Schema{
																																											"name": {
																																												Type:     schema.TypeString,
																																												Required: true,
																																											},
																																											"values": {
																																												Type:     schema.TypeSet,
																																												Elem:     &schema.Schema{Type: schema.TypeString},
																																												Optional: true,
																																											},
																																											"value": {
																																												Type:     schema.TypeString,
																																												Optional: true,
																																											},
																																											"flag": {
																																												Type:     schema.TypeBool,
																																												Optional: true,
																																											},
																																											"type": {
																																												Type:     schema.TypeString,
																																												Optional: true,
																																												Default:  "auto",
																																											},
																																										},
																																									},
																																								},
																																							},
																																						},
																																					},
																																					"behavior": {
																																						Type:     schema.TypeSet,
																																						Optional: true,
																																						Elem: &schema.Resource{
																																							Schema: map[string]*schema.Schema{
																																								"name": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"option": {
																																									Type:     schema.TypeSet,
																																									Optional: true,
																																									Elem: &schema.Resource{
																																										Schema: map[string]*schema.Schema{
																																											"name": {
																																												Type:     schema.TypeString,
																																												Required: true,
																																											},
																																											"values": {
																																												Type:     schema.TypeSet,
																																												Elem:     &schema.Schema{Type: schema.TypeString},
																																												Optional: true,
																																											},
																																											"value": {
																																												Type:     schema.TypeString,
																																												Optional: true,
																																											},
																																											"flag": {
																																												Type:     schema.TypeBool,
																																												Optional: true,
																																											},
																																											"type": {
																																												Type:     schema.TypeString,
																																												Optional: true,
																																												Default:  "auto",
																																											},
																																										},
																																									},
																																								},
																																							},
																																						},
																																					},
																																					"rule": &schema.Schema{
																																						Type:     schema.TypeSet,
																																						Optional: true,
																																						Elem: &schema.Resource{
																																							Schema: map[string]*schema.Schema{
																																								"name": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"comment": {
																																									Type:     schema.TypeString,
																																									Required: true,
																																								},
																																								"criteria_match": {
																																									Type:     schema.TypeString,
																																									Optional: true,
																																									Default:  "all",
																																								},
																																								"criteria": {
																																									Type:     schema.TypeSet,
																																									Optional: true,
																																									Elem: &schema.Resource{
																																										Schema: map[string]*schema.Schema{
																																											"name": {
																																												Type:     schema.TypeString,
																																												Required: true,
																																											},
																																											"option": {
																																												Type:     schema.TypeSet,
																																												Optional: true,
																																												Elem: &schema.Resource{
																																													Schema: map[string]*schema.Schema{
																																														"name": {
																																															Type:     schema.TypeString,
																																															Required: true,
																																														},
																																														"values": {
																																															Type:     schema.TypeSet,
																																															Elem:     &schema.Schema{Type: schema.TypeString},
																																															Optional: true,
																																														},
																																														"value": {
																																															Type:     schema.TypeString,
																																															Optional: true,
																																														},
																																														"flag": {
																																															Type:     schema.TypeBool,
																																															Optional: true,
																																														},
																																														"type": {
																																															Type:     schema.TypeString,
																																															Optional: true,
																																															Default:  "auto",
																																														},
																																													},
																																												},
																																											},
																																										},
																																									},
																																								},
																																								"behavior": {
																																									Type:     schema.TypeSet,
																																									Optional: true,
																																									Elem: &schema.Resource{
																																										Schema: map[string]*schema.Schema{
																																											"name": {
																																												Type:     schema.TypeString,
																																												Required: true,
																																											},
																																											"option": {
																																												Type:     schema.TypeSet,
																																												Optional: true,
																																												Elem: &schema.Resource{
																																													Schema: map[string]*schema.Schema{
																																														"name": {
																																															Type:     schema.TypeString,
																																															Required: true,
																																														},
																																														"values": {
																																															Type:     schema.TypeSet,
																																															Elem:     &schema.Schema{Type: schema.TypeString},
																																															Optional: true,
																																														},
																																														"value": {
																																															Type:     schema.TypeString,
																																															Optional: true,
																																														},
																																														"flag": {
																																															Type:     schema.TypeBool,
																																															Optional: true,
																																														},
																																														"type": {
																																															Type:     schema.TypeString,
																																															Optional: true,
																																															Default:  "auto",
																																														},
																																													},
																																												},
																																											},
																																										},
																																									},
																																								},
																																							},
																																						},
																																					},
																																				},
																																			},
																																		},
																																	},
																																},
																															},
																														},
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}
