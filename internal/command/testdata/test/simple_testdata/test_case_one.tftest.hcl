variables {
  input = "default"
}

# test_run_one runs a partial plan
# run "test_run_one" {
#   command = plan

#   plan_options {
#     target = [
#       test_resource.a
#     ]
#   }

#   assert {
#     condition = test_resource.a.value == "default"
#     error_message = "invalid value"
#   }
# }

# # test_run_two does a complete apply operation
# run "test_run_two" {
#   variables {
#     input = "custom"
#   }

#   assert {
#     condition = test_resource.a.value == run.test_run_one.name
#     error_message = "invalid value"
#   }
# }


run "test1" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test2" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test1.name
    error_message = "description not matching"
  }
}

run "test3" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test2.name
    error_message = "description not matching"
  }
}

run "test4" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}
run "test5" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test4.name
    error_message = "description not matching"
  }
}

run "test6" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test7" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test6.name
    error_message = "description not matching"
  }
}

run "test8" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test7.name
    error_message = "description not matching"
  }
}

run "test9" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test10" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test9.name
    error_message = "description not matching"
  }
}

run "test11" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test10.name
    error_message = "description not matching"
  }
}

run "test12" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test13" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test12.name
    error_message = "description not matching"
  }
}

run "test14" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test13.name
    error_message = "description not matching"
  }
}

run "test15" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test16" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test15.name
    error_message = "description not matching"
  }
}

run "test17" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test16.name
    error_message = "description not matching"
  }
}

run "test18" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test19" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test18.name
    error_message = "description not matching"
  }
}

run "test20" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test19.name
    error_message = "description not matching"
  }
}

run "test21" {
  assert {
    condition = test_resource.a.value == run.test19.name
    error_message = "description not matching"
  }
}

run "test22" {
  assert {
    condition = test_resource.a.value == run.test21.name
    error_message = "description not matching"
  }
}

run "test23" {
  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test24" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test23.name
    error_message = "description not matching"
  }
}

run "test25" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test26" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test25.name
    error_message = "description not matching"
  }
}

run "test27" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test26.name
    error_message = "description not matching"
  }
}

run "test28" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test29" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test28.name
    error_message = "description not matching"
  }
}

run "test30" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test29.name
    error_message = "description not matching"
  }
}

run "test31" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test32" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test31.name
    error_message = "description not matching"
  }
}

run "test33" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test32.name
    error_message = "description not matching"
  }
}

run "test34" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test35" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test34.name
    error_message = "description not matching"
  }
}

run "test36" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test35.name
    error_message = "description not matching"
  }
}

run "test37" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test38" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test37.name
    error_message = "description not matching"
  }
}

run "test39" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test38.name
    error_message = "description not matching"
  }
}

run "test40" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test41" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test40.name
    error_message = "description not matching"
  }
}

run "test42" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test41.name
    error_message = "description not matching"
  }
}

run "test43" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test44" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test43.name
    error_message = "description not matching"
  }
}

run "test45" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test44.name
    error_message = "description not matching"
  }
}

run "test46" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test47" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test46.name
    error_message = "description not matching"
  }
}

run "test48" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test47.name
    error_message = "description not matching"
  }
}

run "test49" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test50" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test49.name
    error_message = "description not matching"
  }
}

run "test51" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test50.name
    error_message = "description not matching"
  }
}

run "test52" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test53" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test52.name
    error_message = "description not matching"
  }
}

run "test54" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test53.name
    error_message = "description not matching"
  }
}

run "test55" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test56" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test55.name
    error_message = "description not matching"
  }
}

run "test57" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test56.name
    error_message = "description not matching"
  }
}

run "test58" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test59" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test58.name
    error_message = "description not matching"
  }
}

run "test60" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test59.name
    error_message = "description not matching"
  }
}

run "test61" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test62" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test61.name
    error_message = "description not matching"
  }
}

run "test63" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test62.name
    error_message = "description not matching"
  }
}

run "test64" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test65" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test64.name
    error_message = "description not matching"
  }
}

run "test66" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test65.name
    error_message = "description not matching"
  }
}

run "test67" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test68" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test67.name
    error_message = "description not matching"
  }
}

run "test69" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test68.name
    error_message = "description not matching"
  }
}

run "test70" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test71" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test70.name
    error_message = "description not matching"
  }
}

run "test72" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test71.name
    error_message = "description not matching"
  }
}

run "test73" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test74" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test73.name
    error_message = "description not matching"
  }
}

run "test75" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test74.name
    error_message = "description not matching"
  }
}

run "test76" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test77" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test76.name
    error_message = "description not matching"
  }
}

run "test78" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test77.name
    error_message = "description not matching"
  }
}

run "test79" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test80" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test79.name
    error_message = "description not matching"
  }
}

run "test81" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test80.name
    error_message = "description not matching"
  }
}

run "test82" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test83" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test82.name
    error_message = "description not matching"
  }
}

run "test84" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test83.name
    error_message = "description not matching"
  }
}

run "test85" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test86" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test85.name
    error_message = "description not matching"
  }
}

run "test87" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test86.name
    error_message = "description not matching"
  }
}

run "test88" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test89" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test88.name
    error_message = "description not matching"
  }
}

run "test90" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test89.name
    error_message = "description not matching"
  }
}

run "test91" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test92" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test91.name
    error_message = "description not matching"
  }
}

run "test93" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test92.name
    error_message = "description not matching"
  }
}

run "test94" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test95" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test94.name
    error_message = "description not matching"
  }
}

run "test96" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test95.name
    error_message = "description not matching"
  }
}

run "test97" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test98" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test97.name
    error_message = "description not matching"
  }
}

run "test99" {
  command = plan

  assert {
    condition = test_resource.a.value == run.test98.name
    error_message = "description not matching"
  }
}

run "test100" {
  command = plan

  assert {
    condition = test_resource.a.value == "default"
    error_message = "description not matching"
  }
}