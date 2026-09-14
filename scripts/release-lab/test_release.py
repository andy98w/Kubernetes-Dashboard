import unittest
from release import evaluate, route_patch, Lab

class ReleaseTests(unittest.TestCase):
    def test_missing_measurements_fail_closed(self):
        self.assertFalse(evaluate([])['pass'])
        self.assertFalse(evaluate([{'status':200,'seconds':float('nan')}]*20)['pass'])
    def test_ready_but_failing_application_is_rejected(self):
        self.assertFalse(evaluate([{'status':503,'seconds':.01}]*20)['pass'])
        self.assertFalse(evaluate([{'status':0,'seconds':2}]*20)['pass'])
    def test_fast_baseline_rejects_slow_candidate(self):
        baseline=evaluate([{'status':200,'seconds':.01}]*20)
        self.assertTrue(baseline['pass'])
        self.assertFalse(evaluate([{'status':200,'seconds':.2}]*20,baseline)['pass'])
    def test_small_error_budget_is_bounded(self):
        samples=[{'status':200,'seconds':.01}]*19+[{'status':503,'seconds':.01}]
        self.assertTrue(evaluate(samples)['pass'])
        self.assertFalse(evaluate(samples[:-2]+[{'status':503,'seconds':.01}]*2)['pass'])
    def test_mutation_guards_observed_version_and_selector(self):
        observed={'metadata':{'resourceVersion':'42'},'spec':{'selector':{'slot':'stable'}}}
        patch=route_patch(observed,'candidate')
        self.assertEqual(patch[0],{'op':'test','path':'/metadata/resourceVersion','value':'42'})
        self.assertEqual(patch[1]['value'],{'slot':'stable'})
    def test_non_lab_context_is_refused(self):
        with self.assertRaises(ValueError): Lab('production','/tmp/unused')

if __name__=='__main__': unittest.main()
