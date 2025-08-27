package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift/cluster-monitoring-operator/pkg/client"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func GetServiceAccountToken(cmoClient *client.Client, namespace, name string) (string, error) {
	var (
		ctx             = context.Background()
		token           string
		tokenExpiration = time.Hour * 12
		expirationTime  = metav1.NewTime(time.Now().Add(tokenExpiration))
	)
	err := Poll(time.Second, time.Minute, func() error {

		tokenReq, err := cmoClient.KubernetesInterface().CoreV1().ServiceAccounts(namespace).CreateToken(
			ctx,
			name,
			&authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: ptr.To(int64((tokenExpiration + time.Minute) / time.Second)),
				},
			},
			metav1.CreateOptions{},
		)
		if err != nil {
			return err
		}

		if tokenReq.Status.ExpirationTimestamp.Before(&expirationTime) {
			return fmt.Errorf("expiration too short: %v < %v", tokenReq.Status.ExpirationTimestamp, expirationTime)
		}

		token = tokenReq.Status.Token
		return nil
	})

	return token, err
}
