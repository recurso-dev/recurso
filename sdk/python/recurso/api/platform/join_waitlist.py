from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.join_waitlist_body import JoinWaitlistBody
from ...models.join_waitlist_response_200 import JoinWaitlistResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: JoinWaitlistBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/waitlist",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | JoinWaitlistResponse200 | None:
    if response.status_code == 200:
        response_200 = JoinWaitlistResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | JoinWaitlistResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JoinWaitlistBody,
) -> Response[Error | JoinWaitlistResponse200]:
    """Join the Recurso Cloud waitlist

     Public, rate-limited demand capture used by the marketing site. Duplicate emails are deduplicated
    silently; the response never reveals whether an email was already on the list. The `website` field
    is a honeypot — leave it out.

    Args:
        body (JoinWaitlistBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | JoinWaitlistResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: JoinWaitlistBody,
) -> Error | JoinWaitlistResponse200 | None:
    """Join the Recurso Cloud waitlist

     Public, rate-limited demand capture used by the marketing site. Duplicate emails are deduplicated
    silently; the response never reveals whether an email was already on the list. The `website` field
    is a honeypot — leave it out.

    Args:
        body (JoinWaitlistBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | JoinWaitlistResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JoinWaitlistBody,
) -> Response[Error | JoinWaitlistResponse200]:
    """Join the Recurso Cloud waitlist

     Public, rate-limited demand capture used by the marketing site. Duplicate emails are deduplicated
    silently; the response never reveals whether an email was already on the list. The `website` field
    is a honeypot — leave it out.

    Args:
        body (JoinWaitlistBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | JoinWaitlistResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: JoinWaitlistBody,
) -> Error | JoinWaitlistResponse200 | None:
    """Join the Recurso Cloud waitlist

     Public, rate-limited demand capture used by the marketing site. Duplicate emails are deduplicated
    silently; the response never reveals whether an email was already on the list. The `website` field
    is a honeypot — leave it out.

    Args:
        body (JoinWaitlistBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | JoinWaitlistResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
