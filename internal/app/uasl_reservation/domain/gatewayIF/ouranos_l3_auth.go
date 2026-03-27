package gatewayIF

import "context"

type OuranosL3AuthGatewayIF interface {
	GetAccessToken(ctx context.Context) (string, error)

	IntrospectToken(ctx context.Context, accessToken string) error
}
